# Source code modified by: Jan Tache
#
# Base source code:
# https://gist.github.com/mafayaz/faf938a896357c3a4c9d6da27edcff08
#
# Usage: python <filename> <port number> <IP address>
# Very simple webserver for serving up two files: web.html and svg.svg

from BaseHTTPServer import BaseHTTPRequestHandler,HTTPServer
from SocketServer import ThreadingMixIn
import threading
import argparse
import re
import cgi
import json

class HTTPRequestHandler(BaseHTTPRequestHandler):
    # Code to handle HTTP POST
    def do_GET(s):
        if s.path == '/svg.svg':
            with open('svg.svg', 'r') as f:
                s.wfile.write(f.read())
            return

        s.send_response(200)
        s.send_header("Content-type", "text/html")
        s.end_headers()
        with open('web.html', 'r') as f:
            s.wfile.write(f.read())

# Rest of source code was unmodified from the original; setting up HTTP server
class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    allow_reuse_address = True

    def shutdown(self):
        self.socket.close()
        HTTPServer.shutdown(self)

class SimpleHttpServer():
    def __init__(self, ip, port):
        self.server = ThreadedHTTPServer((ip,port), HTTPRequestHandler)

    def start(self):
        self.server_thread = threading.Thread(target=self.server.serve_forever)
        self.server_thread.daemon = True
        self.server_thread.start()

    def waitForThread(self):
        self.server_thread.join()

    def addRecord(self, recordID, jsonEncodedRecord):
        LocalData.records[recordID] = jsonEncodedRecord

    def stop(self):
        self.server.shutdown()
        self.waitForThread()

if __name__=='__main__':
    parser = argparse.ArgumentParser(description='HTTP Server')
    parser.add_argument('port', type=int, help='Listening port for HTTP Server')
    parser.add_argument('ip', help='HTTP Server IP')
    args = parser.parse_args()

    server = SimpleHttpServer(args.ip, args.port)
    print 'HTTP Server Running...........'
    server.start()
    server.waitForThread()
