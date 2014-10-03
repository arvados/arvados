import os
import glob
import stat

class SubstitutionError(Exception):
    pass

def search(c):
    DEFAULT = 0
    DOLLAR = 1

    i = 0
    state = DEFAULT
    start = None
    depth = 0
    while i < len(c):
        if c[i] == '\\':
            i += 1
        elif state == DEFAULT:
            if c[i] == '$':
                state = DOLLAR
                if depth == 0:
                    start = i
            elif c[i] == ')':
                if depth == 1:
                    return [start, i]
                if depth > 0:
                    depth -= 1
        elif state == DOLLAR:
            if c[i] == '(':
                depth += 1
            state = DEFAULT
        i += 1
    if depth != 0:
        raise SubstitutionError("Substitution error, mismatched parentheses {}".format(c))
    return None

def sub_file(v):
    path = os.path.join(os.environ['TASK_KEEPMOUNT'], v)
    st = os.stat(path)
    if st and stat.S_ISREG(st.st_mode):
        return path
    else:
        raise SubstitutionError("$(file {}) is not accessable or is not a regular file".format(path))

def sub_dir(v):
    d = os.path.dirname(v)
    if d == '':
        d = v
    path = os.path.join(os.environ['TASK_KEEPMOUNT'], d)
    st = os.stat(path)
    if st and stat.S_ISDIR(st.st_mode):
        return path
    else:
        raise SubstitutionError("$(dir {}) is not accessable or is not a directory".format(path))

def sub_basename(v):
    return os.path.splitext(os.path.basename(v))[0]

def sub_glob(v):
    l = glob.glob(v)
    if len(l) == 0:
        raise SubstitutionError("$(glob {}) no match fonud".format(v))
    else:
        return l[0]

default_subs = {"file ": sub_file,
                "dir ": sub_dir,
                "basename ": sub_basename,
                "glob ": sub_glob}

def do_substitution(p, c, subs=default_subs):
    while True:
        m = search(c)
        if m is None:
            return c

        v = do_substitution(p, c[m[0]+2 : m[1]])
        var = True
        for sub in subs:
            if v.startswith(sub):
                r = subs[sub](v[len(sub):])
                var = False
                break
        if var:
            if v in p:
                r = p[v]
            else:
                raise SubstitutionError("Unknown variable or function '%s' while performing substitution on '%s'" % (v, c))
            if r is None:
                raise SubstitutionError("Substitution for '%s' is null while performing substitution on '%s'" % (v, c))
            if not isinstance(r, basestring):
                raise SubstitutionError("Substitution for '%s' must be a string while performing substitution on '%s'" % (v, c))

        c = c[:m[0]] + r + c[m[1]+1:]
