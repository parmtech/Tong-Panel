import socket
import sys
import unittest

from smoke_noop_worker import check_port_available


class WorkerPortGuardTests(unittest.TestCase):
    def test_rejects_live_listener(self):
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            listener.listen()
            with self.assertRaises(RuntimeError):
                check_port_available(listener.getsockname()[1])

    @unittest.skipUnless(sys.platform.startswith("linux"), "Linux listener reuse contract")
    def test_allows_reuse_after_server_closes_connection(self):
        with socket.socket() as listener:
            listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
            listener.listen()
            with socket.create_connection(("127.0.0.1", port), timeout=1) as client:
                accepted, _ = listener.accept()
                accepted.close()  # Server active-close creates TIME_WAIT.
                self.assertEqual(client.recv(1), b"")
        check_port_available(port)


if __name__ == "__main__":
    unittest.main()
