results <- devtools::test()
any_error <- any(as.data.frame(results)$error)
if (any_error) {
  q("no", 1)
} else {
  q("no", 0)
}
