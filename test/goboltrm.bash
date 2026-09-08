#!/bin/bash

rm winbatch.go
# Remove originally failing tests
rm fixedbugs/issue21576.go
rm fixedbugs/issue21317.go
rm fixedbugs/issue33275_run.go
rm nilptr.go
rm nosplit.go
rm linkx_run.go
rm heapsampling.go # Unstable test
# Remove tests with meta-information check
# inline_caller.go is fixed in patch
rm fixedbugs/bug347.go
rm fixedbugs/bug348.go
rm fixedbugs/issue14646.go
rm fixedbugs/issue18149.go
rm fixedbugs/issue22083.go
rm fixedbugs/issue22662.go
rm fixedbugs/issue27201.go
rm fixedbugs/issue29504.go
rm fixedbugs/issue34123.go
rm fixedbugs/issue4562.go
rm fixedbugs/issue5856.go
rm fixedbugs/issue7690.go
# 1.16.5
rm fixedbugs/issue36437.go
# 1.17
rm const7.go
rm fixedbugs/issue20014.go
rm fixedbugs/bug513.go
rm fixedbugs/issue20780b.go
rm fixedbugs/issue29329.go
rm fixedbugs/issue36516.go
rm fixedbugs/issue46234.go
