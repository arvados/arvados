import json
import sys

f = open(sys.argv[1], "r+")
j = json.load(f)
j["Clusters"][sys.argv[2]]["Login"] = {"LoginCluster": sys.argv[3]}
for r in j["Clusters"][sys.argv[2]]["RemoteClusters"]:
    j["Clusters"][sys.argv[2]]["RemoteClusters"][r]["Insecure"] = True
f.seek(0)
json.dump(j, f)
