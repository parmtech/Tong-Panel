# Unsigned developer Debian package structural verification

**Status: disposable packaging QA only. No signed release or supported installer.**

The checked-in `scripts/build_deb.sh` builds a deliberately unsigned, loopback-only developer `.deb`. The earlier package test listed the data payload with `dpkg-deb --contents`, but that listing does **not** contain the Debian *control archive*, including privileged `preinst` and other maintainer hooks. A checksum of an unsigned artifact does not authenticate its publisher.

`scripts/verify_dev_deb.py` now checks both control and data tar streams **without extracting or installing either**. It permits exactly the developer package's reviewed `control`, `postinst`, `prerm`, `postrm`, the two expected executable binaries, the systemd service, the DO_NOT_INSTALL warning and the license. It rejects unknown paths, extra package hooks, links, nonregular files, duplicate entries, unsafe modes/owners, changed bytes, missing files, inconsistent versions and oversized package inputs. File bytes are compared with the current source fixture and exact compiler-built binaries supplied by the package lab. This check is a development consistency assertion, not a proof that those source files are safe.

## Reproduce on an amd64 Debian-capable disposable runner

```sh
python3 -m unittest discover -s scripts -p 'test_verify_dev_deb.py' -v
python3 scripts/verify_dev_deb.py /path/to/tong-panel-dev-v1.deb 0.0.0~dev.1 /path/to/tongd /path/to/tongctl
python3 scripts/verify_dev_deb.py /path/to/tong-panel-dev-v2.deb 0.0.0~dev.2 /path/to/tongd /path/to/tongctl
```

The nine real `dpkg-deb` fixture tests include one valid package, injected `preinst`, altered `postinst`, extra payload file, replaced and mismatched binaries, symlink, setuid mode, wrong version and missing license data. `scripts/test_deb_package.sh` runs the negative suite and checks both actual Go-built package versions before the VM workflow transfers/install-tests those unsigned fixtures. The clean-VM CI is also triggered by changes to the verifier/tests.

The verifier depends on the working tree's trusted source and binaries. It is **not a signature validator, supply-chain provenance check, hostile-input parser security certification or general-purpose `.deb` auditor**. The disposable VM smoke may exercise installation as part of existing isolated CI; this check never grants production installation. ADR-008 publisher-signing key custody, immutable signed releases, verified bootstrap, adversarial recovery and exact supported distro approval remain open. See [P1 security acceptance](P1_SECURITY_ACCEPTANCE_2026-11-13.md), [master tracker](https://github.com/parmtech/Tong-Panel/issues/172) and [integration PR](https://github.com/parmtech/Tong-Panel/pull/171).
