import sys
import csv

merged = open("merged.csv", "wt")

wroteheader = False
for s in sys.argv[1:]:
    f = open(s, "rt")
    header = next(f)
    if not wroteheader:
        merged.write(header)
        wroteheader = True
    for l in f:
        merged.write(l)
    f.close()
