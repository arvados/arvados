from multiprocessing import Process
import os
import subprocess
import sys
import prof

def fn(n):
    return "file%i" % n

def createfiles(d, n):
    for j in xrange(1, 5):
        print "Starting small file %s %i, %i" % (d, n, j)
        if d:
            os.mkdir(d)
            ld = os.listdir('.')
            if d not in ld:
                print "ERROR %s missing" % d
            os.chdir(d)

        for i in xrange(n, n+10):
            with open(fn(i), "w") as f:
                f.write(fn(i))

        ld = os.listdir('.')
        for i in xrange(n, n+10):
            if fn(i) not in ld:
                print "ERROR %s missing" % fn(i)

        for i in xrange(n, n+10):
            with open(fn(i), "r") as f:
                if f.read() != fn(i):
                    print "ERROR %s doesn't have expected contents" % fn(i)

        for i in xrange(n, n+10):
            os.remove(fn(i))

        ld = os.listdir('.')
        for i in xrange(n, n+10):
            if fn(i) in ld:
                print "ERROR %s should have been removed" % fn(i)

        if d:
            os.chdir('..')
            os.rmdir(d)
            ld = os.listdir('.')
            if d in ld:
                print "ERROR %s should have been removed" % d


def createbigfile(d, n):
    for j in xrange(1, 5):
        print "Starting big file %s %i, %i" % (d, n, j)
        i = n
        if d:
            os.mkdir(d)
            ld = os.listdir('.')
            if d not in ld:
                print "ERROR %s missing" % d
            os.chdir(d)

        with open(fn(i), "w") as f:
            for j in xrange(0, 1000):
                f.write((str(j) + fn(i)) * 10000)

        ld = os.listdir('.')
        if fn(i) not in ld:
            print "ERROR %s missing" % fn(i)

        with open(fn(i), "r") as f:
            for j in xrange(0, 1000):
                expect = (str(j) + fn(i)) * 10000
                if f.read(len(expect)) != expect:
                    print "ERROR %s doesn't have expected contents" % fn(i)

        os.remove(fn(i))

        ld = os.listdir('.')
        if fn(i) in ld:
            print "ERROR %s should have been removed" % fn(i)

        if d:
            os.chdir('..')
            os.rmdir(d)
            ld = os.listdir('.')
            if d in ld:
                print "ERROR %s should have been removed" % d

def do_ls():
    with open("/dev/null", "w") as nul:
        for j in xrange(1, 50):
            subprocess.call(["ls", "-l"], stdout=nul, stderr=nul)

def runit(target, indir):
    procs = []
    for n in xrange(0, 20):
        if indir:
            p = Process(target=target, args=("dir%i" % n, n*10,))
        else:
            p = Process(target=target, args=("", n*10,))
        p.start()
        procs.append(p)

    p = Process(target=do_ls, args=())
    p.start()
    procs.append(p)

    for p in procs:
        p.join()

    if os.listdir('.'):
        print "ERROR there are left over files in the directory"


if __name__ == '__main__':
    if os.listdir('.'):
        print "ERROR starting directory is not empty"
        sys.exit()

    print "Single directory small files"
    with prof.CountTime():
        runit(createfiles, False)

    print "Separate directories small files"
    with prof.CountTime():
        runit(createfiles, True)

    print "Single directory large files"
    with prof.CountTime():
        runit(createbigfile, False)

    print "Separate directories large files"
    with prof.CountTime():
        runit(createbigfile, True)
