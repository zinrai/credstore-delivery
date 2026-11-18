#!/usr/bin/env python3
"""Serve the socket contract of credstore-delivery from SECRETS below.

Listens on ./store.sock. For trying credstore-delivery only: the values
sit in this file.
"""

import http.server
import os
import socketserver

SOCKET = "store.sock"
SECRETS = {
    "web/db-password": b"example-password",
    "shared/api-token": b"example-token\n",
}


class Handler(http.server.BaseHTTPRequestHandler):
    # No name check: a lookup in SECRETS reaches nothing outside it, and
    # a name it does not hold is a 404.
    def do_GET(self):
        value = SECRETS.get(self.path.removeprefix("/"))
        if value is None:
            self.send_error(404)
            return
        self.send_response(200)
        self.send_header("Content-Length", str(len(value)))
        self.end_headers()
        self.wfile.write(value)

    # The socket path, not the client's address: a Unix socket peer has none.
    def address_string(self):
        return SOCKET


class Server(socketserver.ThreadingMixIn, socketserver.UnixStreamServer):
    pass


# umask, not chmod after bind: the socket is never open to others.
os.umask(0o177)
Server(SOCKET, Handler).serve_forever()
