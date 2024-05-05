from http.server import BaseHTTPRequestHandler, HTTPServer
import json
from sklearn.cluster import SpectralClustering
import numpy as np

class MyHTTPServer(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        decoded_post_data = post_data.decode('utf-8')
        
        # 处理接收到的数据
        received_data = json.loads(decoded_post_data)

        response_data = {'message': 'Received data successfully', 'data': received_data,'code' : 501}

        #获取对应的值
        rows_value = received_data.get("rows")
        row = received_data.get("row") #需要聚类的数量
        if rows_value is not None and row is not None:
            print(rows_value)
            numbers_array = [int(num_str) for num_str in rows_value.split(",")]
            # 打印结果
            print(numbers_array)
            clustered_data = self.spec(numbers_array, int(row))
            # 将 numpy.int32 转换为 Python 内置整数类型
            clustered_data_converted = {}
            for key, value in clustered_data.items():
                clustered_data_converted[int(key)] = ','.join(map(str, value))
            response_data['result'] = clustered_data_converted
            response_data['code'] = 200
        
        print(response_data)
        # 发送HTTP响应
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(response_data).encode('utf-8'))

    # k是聚类数量 numbers_array是删选后的行号数组
    def spec(self, numbers_array, k):
        # 从文件中加载数据
        data = np.loadtxt('/root/flame2.txt')

        # 提取特征和标签
        X = data[numbers_array, :3]  # 使用 numbers_array 数组作为索引，选择对应行的前三列作为特征

        # 创建谱聚类的实例并进行聚类
        sc = SpectralClustering(n_clusters=k)
        y_pred = sc.fit_predict(X)

        # 创建一个字典来存储相同类别的数据
        clustered_data = {}

        # 将每个数据点添加到相应的类别中
        for i, label in enumerate(y_pred):
            data_point = numbers_array[i]
            if label not in clustered_data:
                clustered_data[label] = []
            clustered_data[label].append(data_point)

        # 打印每个类别中的数据点
        print("每个类别中的数据点:")
        for label, data_points in clustered_data.items():
            print("类别 {}: {}".format(label, data_points))
        return clustered_data

def run(server_class=HTTPServer, handler_class=MyHTTPServer, port=10002):
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)
    print('Starting Python HTTP server...')
    httpd.serve_forever()

if __name__ == "__main__":
    run()