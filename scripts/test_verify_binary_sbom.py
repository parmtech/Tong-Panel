import base64
import copy
import json
import unittest

from verify_binary_sbom import InvalidSBOM, unique_pairs, verify_document


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


if __name__ == "__main__":
    unittest.main()
