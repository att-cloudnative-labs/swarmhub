from prometheus_client import start_http_server, Summary, Counter
import requests, time, os

REQUEST_TIME = Summary('request_processing_seconds', 'Time spent processing request')
c = Counter('my_requests_total', 'HTTP Failures', ['status'])

# Decorate function with metric.
@REQUEST_TIME.time()
def process_request():
    r = requests.get(url)
    c.labels(r.status_code).inc()
    

if __name__ == '__main__':
    # Start up the server to expose the metrics.
    start_http_server(8080)
    # Generate some requests.
    while True:
        url = os.getenv('PING_URL', None)
        if url == None:
            print('The PING_URL was not set!')
        else:
            process_request()
        time.sleep(1)
