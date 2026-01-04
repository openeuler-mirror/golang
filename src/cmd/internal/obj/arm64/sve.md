# Supported SVE instructions

The following instructions are supported:

| Name                          | Instruction (in Arm style)                      | As                | Case |
|-------------------------------|-------------------------------------------------|-------------------|------|
| ADD (immediate)               | `ADD <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`   | ZADD              | 3    |
| ADD (vectors, predicated)     | `ADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`    | ZADD              | 2    |
| ADD (vectors, unpredicated)   | `ADD <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`              | ZADD              | 1    |
| AND (immediate)               | `AND <Zdn>.<T>, <Zdn>.<T>, #<const>`            | ZAND              | 5    |
| AND (predicates)              | `AND <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`            | PAND              | 6    |
| AND (vectors, predicated)     | `AND <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`    | ZAND              | 2    |
| AND (vectors, unpredicated)   | `AND <Zd>.D, <Zn>.D, <Zm>.D`                    | ZAND              | 4    |
| ANDS                          | `ANDS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`           | PANDS             | 6    |
| ASR (immediate, predicated)   | `ASR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`    | ZASR              | 15   |
| ASRD                          | `ASRD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`   | ZASRD             | 15   |
| BIC (immediate)               | `BIC <Zdn>.<T>, <Zdn>.<T>, #<const>`            | ZBIC              | 5    |
| BIC (predicates)              | `BIC <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`            | PBIC              | 6    |
| BIC (vectors, predicated)     | `BIC <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`    | ZBIC              | 2    |
| BIC (vectors, unpredicated)   | `BIC <Zd>.D, <Zn>.D, <Zm>.D`                    | ZBIC              | 4    |
| BICS                          | `BICS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`           | PBICS             | 6    |
| CMPEQ (immediate)             | `CMPEQ <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPEQ            | 13   |
| CMPEQ (vectors)               | `CMPEQ <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPEQ            | 12   |
| CMPEQ (wide elements)         | `CMPEQ <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPEQ            | 12   |
| CMPGT (immediate)             | `CMPGT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPGT            | 13   |
| CMPGT (vectors)               | `CMPGT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPGT            | 12   |
| CMPGT (wide elements)         | `CMPGT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPGT            | 12   |
| CMPGE (immediate)             | `CMPGE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPGE            | 13   |
| CMPGE (vectors)               | `CMPGE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPGE            | 12   |
| CMPGE (wide elements)         | `CMPGE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPGE            | 12   |
| CMPHI (immediate)             | `CMPHI <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPHI            | 13   |
| CMPHI (vectors)               | `CMPHI <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPHI            | 12   |
| CMPHI (wide elements)         | `CMPHI <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPHI            | 12   |
| CMPHS (immediate)             | `CMPHS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPHS            | 13   |
| CMPHS (vectors)               | `CMPHS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPHS            | 12   |
| CMPHS (wide elements)         | `CMPHS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPHS            | 12   |
| CMPLE (immediate)             | `CMPLE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPLE            | 13   |
| CMPLE (vectors)               | `CMPLE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPLE            | 12   |
| CMPLE (wide elements)         | `CMPLE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPLE            | 12   |
| CMPLO (immediate)             | `CMPLO <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPLO            | 13   |
| CMPLO (vectors)               | `CMPLO <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPLO            | 12   |
| CMPLO (wide elements)         | `CMPLO <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPLO            | 12   |
| CMPLS (immediate)             | `CMPLS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPLS            | 13   |
| CMPLS (vectors)               | `CMPLS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPLS            | 12   |
| CMPLS (wide elements)         | `CMPLS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPLS            | 12   |
| CMPLT (immediate)             | `CMPLT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPLT            | 13   |
| CMPLT (vectors)               | `CMPLT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPLT            | 12   |
| CMPLT (wide elements)         | `CMPLT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPLT            | 12   |
| CMPNE (immediate)             | `CMPNE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`      | ZCMPNE            | 13   |
| CMPNE (vectors)               | `CMPNE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`    | ZCMPNE            | 12   |
| CMPNE (wide elements)         | `CMPNE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`      | ZCMPNE            | 12   |
| DUP (immediate)               | `DUP <Zd>.<T>, #<imm>{, <shift>}`               | ZDUP              | 9    |
| DUP (indexed)                 | `DUP <Zd>.<T>, <Zn>.<T>[<imm>]`                 | ZDUP              | 8    |
| DUP (scalar)                  | `DUP <Zd>.<T>, <R><n\|SP>`                      | ZDUP              | 7    |
| EON                           | `EON <Zdn>.<T>, <Zdn>.<T>, #<const>`            | ZEON              | 5    |
| EOR (immediate)               | `EOR <Zdn>.<T>, <Zdn>.<T>, #<const>`            | ZEOR              | 5    |
| EOR (predicates)              | `EOR <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`            | PEOR              | 6    |
| EOR (vectors, predicated)     | `EOR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`    | ZEOR              | 2    |
| EOR (vectors, unpredicated)   | `EOR <Zd>.D, <Zn>.D, <Zm>.D`                    | ZEOR              | 4    |
| EORS                          | `EORS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`           | PEORS             | 6    |
| LSL (immediate, predicated)   | `LSL <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`    | ZLSL              | 15   |
| LSR (immediate, predicated)   | `LSR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`    | ZLSR              | 15   |
| NAND                          | `NAND <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`           | PNAND             | 6    |
| NANDS                         | `NANDS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`          | PNANDS            | 6    |
| NOR                           | `NOR <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`            | PNOR              | 6    |
| NORS                          | `NORS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`           | PNORS             | 6    |
| ORN (immediate)               | `ORN <Zdn>.<T>, <Zdn>.<T>, #<const>`            | ZORN              | 5    |
| ORN (predicates)              | `ORN <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`            | PORN              | 6    |
| ORNS                          | `ORNS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`           | PORNS             | 6    |
| ORR (immediate)               | `ORR <Zdn>.<T>, <Zdn>.<T>, #<const>`            | ZORR              | 5    |
| ORR (predicates)              | `ORR <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`            | ZORR              | 6    |
| ORR (vectors, predicated)     | `ORR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`    | ZORR              | 2    |
| ORR (vectors, unpredicated)   | `ORR <Zd>.D, <Zn>.D, <Zm>.D`                    | ZORR              | 4    |
| ORRS                          | `ORRS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`           | PORRS             | 6    |
| SEL (predicates)              | `SEL <Pd>.B, <Pg>, <Pn>.B, <Pm>.B`              | PSEL              | 6    |
| SEL (vectors)                 | `SEL <Zd>.<T>, <Pv>, <Zn>.<T>, <Zm>.<T>`        | ZSEL              | 14   |
| SQADD (immediate)             | `SQADD <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}` | ZSQADD            | 3    |
| SQADD (vectors, predicated)   | `SQADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`  | ZSQADD            | 2    |
| SQADD (vectors, unpredicated) | `SQADD <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`            | ZSQADD            | 1    |
| SQSHL (immediate)             | `SQSHL <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`  | ZSQSHL            | 15   |
| SQSHLU                        | `SQSHLU <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>` | ZSQSHLU           | 15   |
| SQSUB (immediate)             | `SQSUB <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}` | ZSQSUB            | 3    |
| SQSUB (vectors, predicated)   | `SQSUB <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`  | ZSQSUB            | 2    |
| SQSUB (vectors, unpredicated) | `SQSUB <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`            | ZSQSUB            | 1    |
| SQSUBR                        | `SQSUBR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>` | ZSQSUBR           | 2    |
| SRSHR                         | `SRSHR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`  | ZSRSHR            | 15   |
| SUB (immediate)               | `SUB <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`   | ZSUB              | 3    |
| SUB (vectors, predicated)     | `SUB <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`    | ZSUB              | 2    |
| SUB (vectors, unpredicated)   | `SUB <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`              | ZSUB              | 1    |
| SUBR (immediate)              | `SUBR <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`  | ZSUBR             | 3    |
| SUBR (vectors)                | `SUBR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`   | ZSUBR             | 2    |
| SUQADD                        | `SUQADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>` | ZSUQADD           | 2    |
| UQADD (immediate)             | `UQADD <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}` | ZUQADD            | 3    |
| UQADD (vectors, predicated)   | `UQADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`  | ZUQADD            | 2    |
| UQADD (vectors, unpredicated) | `UQADD <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`            | ZUQADD            | 1    |
| UQSHL (immediate)             | `UQSHL <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`  | ZUQSHL            | 15   |
| UQSUB (immediate)             | `UQSUB <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}` | ZUQSUB            | 3    |
| UQSUB (vectors, predicated)   | `UQSUB <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`  | ZUQSUB            | 2    |
| UQSUB (vectors, unpredicated) | `UQSUB <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`            | ZUQSUB            | 1    |
| UQSUBR                        | `UQSUBR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>` | ZUQSUBR           | 2    |
| URSHR                         | `URSHR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`  | ZURSHR            | 15   |
| USQADD                        | `USQADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>` | ZUSQADD           | 2    |
| WHILEGE (predicate pair)      | `WHILEGE { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`  | WHILEGE           | 11   |
| WHILEGE (predicate)           | `WHILEGE <Pd>.<T>, <R><n>, <R><m>`              | WHILEGE, WHILEGEW | 10   |
| WHILEGT (predicate pair)      | `WHILEGT { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`  | WHILEGT           | 11   |
| WHILEGT (predicate)           | `WHILEGT <Pd>.<T>, <R><n>, <R><m>`              | WHILEGT, WHILEGTW | 10   |
| WHILEHI (predicate pair)      | `WHILEHI { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`  | WHILEHI           | 11   |
| WHILEHI (predicate)           | `WHILEHI <Pd>.<T>, <R><n>, <R><m>`              | WHILEHI, WHILEHIW | 10   |
| WHILEHS (predicate pair)      | `WHILEHS { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`  | WHILEHS           | 11   |
| WHILEHS (predicate)           | `WHILEHS <Pd>.<T>, <R><n>, <R><m>`              | WHILEHS, WHILEHSW | 10   |
| WHILELE (predicate pair)      | `WHILELE { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`  | WHILELE           | 11   |
| WHILELE (predicate)           | `WHILELE <Pd>.<T>, <R><n>, <R><m>`              | WHILELE, WHILELEW | 10   |
| WHILELO (predicate pair)      | `WHILELO { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`  | WHILELO           | 11   |
| WHILELO (predicate)           | `WHILELO <Pd>.<T>, <R><n>, <R><m>`              | WHILELO, WHILELOW | 10   |
| WHILELS (predicate pair)      | `WHILELS { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`  | WHILELS           | 11   |
| WHILELS (predicate)           | `WHILELS <Pd>.<T>, <R><n>, <R><m>`              | WHILELS, WHILELSW | 10   |
| WHILELT (predicate pair)      | `WHILELT { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`  | WHILELT           | 11   |
| WHILELT (predicate)           | `WHILELT <Pd>.<T>, <R><n>, <R><m>`              | WHILELT, WHILELTW | 10   |
