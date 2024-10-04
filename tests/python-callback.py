from http.server import BaseHTTPRequestHandler, HTTPServer
import urllib.parse
import json

class SimpleHTTPRequestHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        parsed_url = urllib.parse.urlparse(self.path)
        url_path = parsed_url.path
        query_params = urllib.parse.parse_qs(parsed_url.query)
        
        headers = dict(self.headers)
        
        content_length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(content_length).decode('utf-8')
        
        response = {
            'url': self.path,
            'url_path': url_path,
            'query_params': query_params,
            'headers': headers,
            'body': body
        }

        
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        
        self.wfile.write(json.dumps(response).encode('utf-8'))

        to_return = json.dumps(response, sort_keys=False, indent=2)
        with open('Output.json', 'a') as f:
            f.write(to_return)
            f.write("\n\n")

        print(to_return)
        print("")

def run(server_class=HTTPServer, handler_class=SimpleHTTPRequestHandler, port=2343):
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)
    print(f'Server running on port {port}...')
    httpd.serve_forever()

if __name__ == '__main__':
    run()
