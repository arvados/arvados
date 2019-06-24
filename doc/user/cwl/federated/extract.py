import csv
import sys

select_column = sys.argv[1]
select_values = sys.argv[2]
dataset = sys.argv[3]
cluster = sys.argv[4]

sv = open(select_values, "rt")
selectvals = [s.strip() for s in sv]

print("selectvals", selectvals)

ds = csv.reader(open(dataset, "rt"))
header = next(ds)
print("header is", header)
columnindex = None
for i,v in enumerate(header):
    if v == select_column:
        columnindex = i
if columnindex is None:
    raise Exception("Column %s not found" % select_column)

print("column index", columnindex)

ex = csv.writer(open("extracted.csv", "wt"))
ex.writerow(["cluster"]+list(header))

for row in ds:
    if row[columnindex] in selectvals:
        ex.writerow([cluster]+list(row))
