import base64
import copy
import hashlib
import json
from pathlib import Path
import stat
from tempfile import TemporaryDirectory
from types import SimpleNamespace
import unittest
from unittest.mock import patch

from verify_binary_sbom import InvalidSBOM, unique_pairs, verify_document, verify_files


class BinarySBOMTests(unittest.TestCase):
    def setUp(self):
        self.main = "example.test/panel/cmd/server"
        self.digest = "a" * 64
        self.build = {
            "Path": self.main, "GoVersion": "go1.27.3",
            "Main": {"Path": "example.test/panel", "Version": "(devel)"},
            "Deps": [{"Path": "example.test/library", "Version": "v1.2.3",
                      "Sum": "h1:" + base64.b64encode(bytes(32)).decode()}],
            "Settings": [{"Key": key, "Value": value} for key, value in
                         {"GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "0"}.items()],
        }
        self.document = {
            "bomFormat": "CycloneDX", "specVersion": "1.6", "version": 1,
            "metadata": {
                "component": {"bom-ref": "app", "type": "application", "name": "example.test/panel",
                              "purl": "pkg:golang/example.test/panel?goarch=amd64&goos=linux&type=module#cmd/server"},
                "properties": [{"name": key, "value": value} for key, value in {
                    "cdx:gomod:binary:hash:SHA-256": self.digest,
                    "cdx:gomod:binary:name": "server",
                    "cdx:gomod:build:env:GOOS": "linux",
                    "cdx:gomod:build:env:GOARCH": "amd64",
                    "cdx:gomod:build:env:CGO_ENABLED": "0",
                    "cdx:gomod:build:env:GOVERSION": "go1.27.3",
                }.items()],
            },
            "components": [
                {"bom-ref": "lib", "type": "library", "name": "example.test/library", "version": "v1.2.3",
                 "purl": "pkg:golang/example.test/library@v1.2.3?goarch=amd64&goos=linux&type=module",
                 "hashes": [{"alg": "SHA-256", "content": "0" * 64}]},
                {"bom-ref": "std", "type": "library", "name": "std", "version": "go1.27.3",
                 "purl": "pkg:golang/std@go1.27.3?goarch=amd64&goos=linux&type=module"},
            ],
            "dependencies": [{"ref": "app", "dependsOn": ["lib"]}, {"ref": "lib"}, {"ref": "std"}],
        }

    def verify(self, document=None, build=None, digest=None, main=None):
        verify_document(document or self.document, build or self.build,
                        digest or self.digest, "server", main or self.main)

    def test_accepts_complete_binary_inventory(self):
        self.verify()

    def test_rejects_changed_binary_and_wrong_application(self):
        for kwargs in [{"digest": "b" * 64}, {"main": "example.test/panel/cmd/client"}]:
            with self.subTest(kwargs=kwargs), self.assertRaises(InvalidSBOM):
                self.verify(**kwargs)

    def test_rejects_missing_extra_and_duplicate_modules(self):
        for change in ["missing", "extra", "duplicate"]:
            document = copy.deepcopy(self.document)
            if change == "missing":
                document["components"].pop(0)
            else:
                component = copy.deepcopy(document["components"][0])
                if change == "extra":
                    component.update({"name": "unexpected", "bom-ref": "extra"})
                document["components"].append(component)
            with self.subTest(change=change), self.assertRaises(InvalidSBOM):
                self.verify(document=document)

    def test_rejects_dependency_version_checksum_and_toolchain_tampering(self):
        for field in ["version", "hashes", "stdlib"]:
            document = copy.deepcopy(self.document)
            if field == "stdlib":
                document["components"][1]["version"] = "go1.0"
            elif field == "hashes":
                document["components"][0]["hashes"][0]["content"] = "f" * 64
            else:
                document["components"][0]["version"] = "v9.0.0"
            with self.subTest(field=field), self.assertRaises(InvalidSBOM):
                self.verify(document=document)

    def test_rejects_wrong_target_in_binary_metadata_and_purls(self):
        build = copy.deepcopy(self.build)
        build["Settings"][0]["Value"] = "windows"
        with self.assertRaises(InvalidSBOM):
            self.verify(build=build)
        for component in ["root", "library", "std"]:
            document = copy.deepcopy(self.document)
            item = document["metadata"]["component"] if component == "root" else document["components"][component == "std"]
            item["purl"] = item["purl"].replace("goos=linux", "goos=windows")
            with self.subTest(component=component), self.assertRaises(InvalidSBOM):
                self.verify(document=document)

    def test_rejects_duplicate_properties_and_broken_edges(self):
        document = copy.deepcopy(self.document)
        document["metadata"]["properties"].append(document["metadata"]["properties"][0])
        with self.assertRaises(InvalidSBOM):
            self.verify(document=document)
        for targets in [[], ["lib", "unknown"], ["lib", "lib"]]:
            document = copy.deepcopy(self.document)
            document["dependencies"][0]["dependsOn"] = targets
            with self.subTest(targets=targets), self.assertRaises(InvalidSBOM):
                self.verify(document=document)

    def test_rejects_uninventoried_module_replacements(self):
        build = copy.deepcopy(self.build)
        build["Deps"][0]["Replace"] = {"Path": "../local"}
        with self.assertRaises(InvalidSBOM):
            self.verify(build=build)

    def test_rejects_duplicate_json_members(self):
        with self.assertRaises(InvalidSBOM):
            json.loads('{"components": [], "components": []}', object_pairs_hook=unique_pairs)

    def test_reads_build_metadata_without_executing_binary(self):
        with TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            binary = root / "server"
            sbom = root / "server.cdx.json"
            contents = b"synthetic unexecuted binary fixture"
            binary.write_bytes(contents)

            document = copy.deepcopy(self.document)
            binary_hash = next(
                item for item in document["metadata"]["properties"]
                if item["name"] == "cdx:gomod:binary:hash:SHA-256"
            )
            binary_hash["value"] = hashlib.sha256(contents).hexdigest()
            sbom.write_text(json.dumps(document), encoding="utf-8")

            with patch(
                "verify_binary_sbom.subprocess.run",
                return_value=SimpleNamespace(stdout=json.dumps(self.build)),
            ) as metadata_command:
                verify_files(binary, sbom, self.main)

            metadata_command.assert_called_once_with(
                ["go", "version", "-m", "-json", str(binary)],
                check=True,
                capture_output=True,
                text=True,
                timeout=30,
            )

    def test_rejects_symlink_inputs_before_go_metadata_subprocess(self):
        with TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            binary = root / "tongd"
            sbom = root / "tongd.cdx.json"
            linked_binary = root / "linked-tongd"
            linked_sbom = root / "linked-tongd.cdx.json"
            binary.write_bytes(b"not executed")
            sbom.write_text("{}", encoding="utf-8")

            symlinks = ((linked_binary, binary), (linked_sbom, sbom))
            emulated_links = set()
            for link, target in symlinks:
                try:
                    link.symlink_to(target)
                except (OSError, NotImplementedError):
                    emulated_links.add(link)

            real_lstat = Path.lstat

            def synthetic_lstat(path, *args, **kwargs):
                if path in emulated_links:
                    return SimpleNamespace(st_mode=stat.S_IFLNK | 0o777)
                return real_lstat(path, *args, **kwargs)

            def assert_symlink_paths_rejected():
                for invalid_binary, invalid_sbom in (
                    (linked_binary, sbom),
                    (binary, linked_sbom),
                ):
                    with self.subTest(binary=invalid_binary, sbom=invalid_sbom):
                        with self.assertRaisesRegex(InvalidSBOM, "regular file"):
                            verify_files(invalid_binary, invalid_sbom, self.main)

            with patch("verify_binary_sbom.subprocess.run") as go_version:
                if emulated_links:
                    with patch.object(Path, "lstat", new=synthetic_lstat):
                        assert_symlink_paths_rejected()
                else:
                    assert_symlink_paths_rejected()
                go_version.assert_not_called()

    def test_rejects_directory_inputs_before_go_metadata_subprocess(self):
        with TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            binary = root / "tongd"
            sbom = root / "tongd.cdx.json"
            input_directory = root / "input-directory"
            binary.write_bytes(b"not executed")
            sbom.write_text("{}", encoding="utf-8")
            input_directory.mkdir()

            invalid_pairs = [
                (input_directory, sbom),
                (binary, input_directory),
            ]
            with patch("verify_binary_sbom.subprocess.run") as go_version:
                for invalid_binary, invalid_sbom in invalid_pairs:
                    with self.subTest(binary=invalid_binary, sbom=invalid_sbom):
                        with self.assertRaisesRegex(InvalidSBOM, "regular file"):
                            verify_files(invalid_binary, invalid_sbom, self.main)
                go_version.assert_not_called()

    def test_rejects_oversized_sbom_before_go_metadata_subprocess(self):
        with TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            binary = root / "tongd"
            sbom = root / "tongd.cdx.json"
            binary.write_bytes(b"not executed")
            sbom.write_bytes(b"x" * (4 * 1024 * 1024 + 1))

            with patch("verify_binary_sbom.subprocess.run") as go_version:
                with self.assertRaisesRegex(InvalidSBOM, "SBOM exceeds size bound"):
                    verify_files(binary, sbom, self.main)
                go_version.assert_not_called()


if __name__ == "__main__":
    unittest.main()
