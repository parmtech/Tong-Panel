#!/usr/bin/env python3
"""Verify the developer SBOM's binary/module contract; never execute the binary."""

import base64
import hashlib
import json
from pathlib import Path
import stat
import subprocess
import sys
from urllib.parse import parse_qsl, unquote


class InvalidSBOM(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise InvalidSBOM(message)


def unique_pairs(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate JSON member")
        result[key] = value
    return result


def unique_values(items, key, value):
    result = {}
    for item in items:
        name = item[key]
        require(name not in result, "duplicate SBOM property or checksum")
        result[name] = item[value]
    return result


def verify_purl(component, name, version, subpath=""):
    purl = component["purl"]
    require(purl.startswith("pkg:golang/"), "unexpected package URL type")
    package, _, fragment = purl[len("pkg:golang/"):].partition("#")
    identity, separator, query = package.partition("?")
    qualifiers = unique_pairs(parse_qsl(query, strict_parsing=True))
    expected = name + ("@" + version if version else "")
    require(separator and unquote(identity) == expected and unquote(fragment) == subpath,
            "package URL identity differs from binary")
    require(qualifiers == {"goos": "linux", "goarch": "amd64", "type": "module"},
            "package URL target differs from developer build")


def verify_document(document, build, digest, binary_name, expected_main):
    require(document["bomFormat"] == "CycloneDX" and document["specVersion"] == "1.6",
            "expected CycloneDX 1.6")
    require(document["version"] == 1, "unexpected document version")
    require(build["Path"] == expected_main, "unexpected binary main package")
    require(not build["Main"].get("Replace"), "replaced main module is unsupported")
    main_module = build["Main"]["Path"]
    require(expected_main.startswith(main_module + "/"), "main package is outside module")
    settings = unique_values(build["Settings"], "Key", "Value")
    target = {"GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "0"}
    require(all(settings.get(key) == value for key, value in target.items()),
            "binary is not the Linux amd64 CGO-disabled developer target")
    metadata = document["metadata"]
    properties = unique_values(metadata["properties"], "name", "value")
    require(properties["cdx:gomod:binary:hash:SHA-256"] == digest,
            "SBOM SHA-256 does not match binary bytes")
    require(properties["cdx:gomod:binary:name"] == binary_name, "wrong binary name")
    for key, value in {**target, "GOVERSION": build["GoVersion"]}.items():
        require(properties["cdx:gomod:build:env:" + key] == value,
                "SBOM build setting differs from binary")
    root = metadata["component"]
    require(root["type"] == "application" and root["name"] == main_module,
            "unexpected application component")
    require(isinstance(root["bom-ref"], str) and bool(root["bom-ref"]),
            "missing application reference")
    version = build["Main"].get("Version", "")
    if version == "(devel)":
        version = ""
    require(root.get("version", "") == version, "unexpected main module version")
    verify_purl(root, main_module, version, expected_main[len(main_module) + 1:])
    expected = {}
    for dependency in build.get("Deps", []):
        require(not dependency.get("Replace"), "replaced dependency is unsupported")
        name = dependency["Path"]
        require(name not in expected and name != "std", "duplicate binary dependency")
        expected[name] = dependency
    components = {}
    refs = {root["bom-ref"]}
    for component in document["components"]:
        name, ref = component["name"], component["bom-ref"]
        require(name not in components and ref not in refs and bool(ref),
                "duplicate component name or reference")
        require(component["type"] == "library" and not component.get("components"),
                "unexpected component kind or nesting")
        components[name] = component
        refs.add(ref)
    require(set(components) == set(expected) | {"std"},
            "SBOM module inventory differs from binary build info")
    for name, dependency in expected.items():
        component = components[name]
        require(component["version"] == dependency["Version"], "dependency version mismatch")
        verify_purl(component, name, dependency["Version"])
        checksum = dependency.get("Sum", "")
        require(checksum.startswith("h1:"), "missing authenticated module checksum")
        raw = base64.b64decode(checksum[3:], validate=True)
        require(len(raw) == 32, "invalid module checksum")
        hashes = unique_values(component["hashes"], "alg", "content")
        require(hashes.get("SHA-256") == raw.hex(), "dependency checksum mismatch")
    standard = components["std"]
    require(standard["version"] == build["GoVersion"], "Go standard library version mismatch")
    verify_purl(standard, "std", build["GoVersion"])
    edges = {}
    for dependency in document["dependencies"]:
        ref = dependency["ref"]
        require(ref in refs and ref not in edges, "unknown or duplicate dependency reference")
        targets = dependency.get("dependsOn", [])
        require(len(set(targets)) == len(targets) and set(targets) <= refs,
                "invalid dependency edge")
        edges[ref] = set(targets)
    required_refs = {components[name]["bom-ref"] for name in expected}
    require(required_refs <= edges.get(root["bom-ref"], set()),
            "application dependency edges are incomplete")


def regular_file(path):
    require(stat.S_ISREG(path.lstat().st_mode), "artifact must be a regular file, not a link")


def verify_files(binary, sbom, expected_main):
    regular_file(binary)
    regular_file(sbom)
    require(sbom.stat().st_size <= 4 * 1024 * 1024, "SBOM exceeds size bound")
    document = json.loads(sbom.read_text(encoding="utf-8"), object_pairs_hook=unique_pairs)
    result = subprocess.run(["go", "version", "-m", "-json", str(binary)],
                            check=True, capture_output=True, text=True, timeout=30)
    build = json.loads(result.stdout, object_pairs_hook=unique_pairs)
    checksum = hashlib.sha256()
    with binary.open("rb") as source:
        for block in iter(lambda: source.read(1024 * 1024), b""):
            checksum.update(block)
    verify_document(document, build, checksum.hexdigest(), binary.name, expected_main)


def main(argv):
    if len(argv) != 4:
        print("usage: verify_binary_sbom.py BINARY SBOM EXPECTED_MAIN_PACKAGE", file=sys.stderr)
        return 2
    try:
        verify_files(Path(argv[1]), Path(argv[2]), argv[3])
    except (ValueError, KeyError, TypeError, AttributeError, OSError, subprocess.SubprocessError) as error:
        print("SBOM verification failed: " + str(error), file=sys.stderr)
        return 1
    print("binary-bound SBOM inventory and checksums verified")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
