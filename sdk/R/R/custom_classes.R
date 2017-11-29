#Study(Fudo): For some reason this doesnt seem to work,
#see what seems to be a problem.

setOldClass("characterOrNULL")
setClassUnion("characterOrNULL", c("character", "NULL"))

setClassUnion("listOrNULL", c("list", "NULL"))
setOldClass("listOrNULL")

setOldClass("numericOrNULL")
setClassUnion("numericOrNULL", c("numeric", "NULL"))
