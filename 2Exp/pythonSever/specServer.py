from sklearn.cluster import SpectralClustering
import numpy as np



#k是聚类数量 numbers_array是删选后的行号数组
def spec(numbers_array,k):
    # 从文件中加载数据
    data = np.loadtxt('flame2.txt')

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
