import grpc
from concurrent import futures
import time

import api_pb2
import api_pb2_grpc

class GreeterServicer(api_pb2_grpc.HealthServiceServicer):
    def healthcheck(self, request, context):
        return api_pb2.Empty(
        )

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    api_pb2_grpc.add_HealthServiceServicer_to_server(GreeterServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("Python gRPC server listening on 0.0.0.0:50051")
    try:
        while True:
            time.sleep(86400)
    except KeyboardInterrupt:
        server.stop(0)

if __name__ == '__main__':
    serve()

