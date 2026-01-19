# Supported SVE instructions

The following instructions are supported:

| Name                                          | Instruction (in Arm style)                                                          | As                | Case |
|-----------------------------------------------|-------------------------------------------------------------------------------------|-------------------|------|
| ABS                                           | `ABS <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                    | ZABS              | 17   |
| AESD                                          | `AESD <Zdn>.B, <Zdn>.B, <Zm>.B`                                                     | ZAESD             | 36   |
| AESE                                          | `AESE <Zdn>.B, <Zdn>.B, <Zm>.B`                                                     | ZAESE             | 36   |
| AESIMC                                        | `AESIMC <Zdn>.B, <Zdn>.B`                                                           | ZAESIMC           | 37   |
| AESMC                                         | `AESMC <Zdn>.B, <Zdn>.B`                                                            | ZAESMC            | 37   |
| ADD (immediate)                               | `ADD <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`                                       | ZADD              | 3    |
| ADD (vectors, predicated)                     | `ADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZADD              | 2    |
| ADD (vectors, unpredicated)                   | `ADD <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                  | ZADD              | 1    |
| AND (immediate)                               | `AND <Zdn>.<T>, <Zdn>.<T>, #<const>`                                                | ZAND              | 5    |
| AND (predicates)                              | `AND <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                                | PAND              | 6    |
| AND (vectors, predicated)                     | `AND <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZAND              | 2    |
| AND (vectors, unpredicated)                   | `AND <Zd>.D, <Zn>.D, <Zm>.D`                                                        | ZAND              | 4    |
| ANDS                                          | `ANDS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                               | PANDS             | 6    |
| ASR (immediate, predicated)                   | `ASR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`                                        | ZASR              | 15   |
| ASR (immediate, unpredicated)                 | `ASR <Zd>.<T>, <Zn>.<T>, #<const>`                                                  | ZASR              | 18   |
| ASR (vectors)                                 | `ASR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZASR              | 16   |
| ASR (wide elements, predicated)               | `ASR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.D`                                          | ZASR              | 16   |
| ASR (wide elements, unpredicated)             | `ASR <Zd>.<T>, <Zn>.<T>, <Zm>.D`                                                    | ZASR              | 19   |
| ASRD                                          | `ASRD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`                                       | ZASRD             | 15   |
| ASRR                                          | `ASRR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZASRR             | 16   |
| BIC (immediate)                               | `BIC <Zdn>.<T>, <Zdn>.<T>, #<const>`                                                | ZBIC              | 5    |
| BIC (predicates)                              | `BIC <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                                | PBIC              | 6    |
| BIC (vectors, predicated)                     | `BIC <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZBIC              | 2    |
| BIC (vectors, unpredicated)                   | `BIC <Zd>.D, <Zn>.D, <Zm>.D`                                                        | ZBIC              | 4    |
| BICS                                          | `BICS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                               | PBICS             | 6    |
| BRKA                                          | `BRKA <Pd>.B, <Pg>/<ZM>, <Pn>.B`                                                    | PBRKA             | 31   |
| BRKAS                                         | `BRKAS <Pd>.B, <Pg>/Z, <Pn>.B`                                                      | PBRKAS            | 31   |
| BRKB                                          | `BRKB <Pd>.B, <Pg>/<ZM>, <Pn>.B`                                                    | PBRKB             | 31   |
| BRKBS                                         | `BRKBS <Pd>.B, <Pg>/Z, <Pn>.B`                                                      | PBRKBS            | 31   |
| BRKN                                          | `BRKN <Pdm>.B, <Pg>/Z, <Pn>.B, <Pdm>.B`                                             | PBRKN             | 30   |
| BRKNS                                         | `BRKNS <Pdm>.B, <Pg>/Z, <Pn>.B, <Pdm>.B`                                            | PBRKNS            | 30   |
| BRKPA                                         | `BRKPA <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                              | PBRKPA            | 6    |
| BRKPAS                                        | `BRKPAS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                             | PBRKPAS           | 6    |
| BRKPB                                         | `BRKPB <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                              | PBRKPB            | 6    |
| BRKPBS                                        | `BRKPBS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                             | PBRKPBS           | 6    |
| CLS                                           | `CLS <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                    | ZCLS              | 17   |
| CLZ                                           | `CLZ <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                    | ZCLZ              | 17   |
| CMPEQ (immediate)                             | `CMPEQ <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPEQ            | 13   |
| CMPEQ (vectors)                               | `CMPEQ <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPEQ            | 12   |
| CMPEQ (wide elements)                         | `CMPEQ <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPEQ            | 12   |
| CMPGT (immediate)                             | `CMPGT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPGT            | 13   |
| CMPGT (vectors)                               | `CMPGT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPGT            | 12   |
| CMPGT (wide elements)                         | `CMPGT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPGT            | 12   |
| CMPGE (immediate)                             | `CMPGE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPGE            | 13   |
| CMPGE (vectors)                               | `CMPGE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPGE            | 12   |
| CMPGE (wide elements)                         | `CMPGE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPGE            | 12   |
| CMPHI (immediate)                             | `CMPHI <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPHI            | 13   |
| CMPHI (vectors)                               | `CMPHI <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPHI            | 12   |
| CMPHI (wide elements)                         | `CMPHI <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPHI            | 12   |
| CMPHS (immediate)                             | `CMPHS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPHS            | 13   |
| CMPHS (vectors)                               | `CMPHS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPHS            | 12   |
| CMPHS (wide elements)                         | `CMPHS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPHS            | 12   |
| CMPLE (immediate)                             | `CMPLE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPLE            | 13   |
| CMPLE (vectors)                               | `CMPLE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPLE            | 12   |
| CMPLE (wide elements)                         | `CMPLE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPLE            | 12   |
| CMPLO (immediate)                             | `CMPLO <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPLO            | 13   |
| CMPLO (vectors)                               | `CMPLO <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPLO            | 12   |
| CMPLO (wide elements)                         | `CMPLO <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPLO            | 12   |
| CMPLS (immediate)                             | `CMPLS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPLS            | 13   |
| CMPLS (vectors)                               | `CMPLS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPLS            | 12   |
| CMPLS (wide elements)                         | `CMPLS <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPLS            | 12   |
| CMPLT (immediate)                             | `CMPLT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPLT            | 13   |
| CMPLT (vectors)                               | `CMPLT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPLT            | 12   |
| CMPLT (wide elements)                         | `CMPLT <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPLT            | 12   |
| CMPNE (immediate)                             | `CMPNE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, #<imm>`                                          | ZCMPNE            | 13   |
| CMPNE (vectors)                               | `CMPNE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.<T>`                                        | ZCMPNE            | 12   |
| CMPNE (wide elements)                         | `CMPNE <Pd>.<T>, <Pg>/Z, <Zn>.<T>, <Zm>.D`                                          | ZCMPNE            | 12   |
| CNOT                                          | `CNOT <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                   | ZCNOT             | 17   |
| CNT                                           | `CNT <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                    | ZCNT              | 17   |
| DUP (immediate)                               | `DUP <Zd>.<T>, #<imm>{, <shift>}`                                                   | ZDUP              | 3    |
| DUP (indexed)                                 | `DUP <Zd>.<T>, <Zn>.<T>[<imm>]`                                                     | ZDUP              | 8    |
| DUP (scalar)                                  | `DUP <Zd>.<T>, <R><n\|SP>`                                                          | ZDUP              | 7    |
| EON                                           | `EON <Zdn>.<T>, <Zdn>.<T>, #<const>`                                                | ZEON              | 5    |
| EOR (immediate)                               | `EOR <Zdn>.<T>, <Zdn>.<T>, #<const>`                                                | ZEOR              | 5    |
| EOR (predicates)                              | `EOR <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                                | PEOR              | 6    |
| EOR (vectors, predicated)                     | `EOR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZEOR              | 2    |
| EOR (vectors, unpredicated)                   | `EOR <Zd>.D, <Zn>.D, <Zm>.D`                                                        | ZEOR              | 4    |
| EORS                                          | `EORS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                               | PEORS             | 6    |
| FABS                                          | `FABS <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                   | ZFABS             | 17   |
| FNEG                                          | `FNEG <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                   | ZFNEG             | 17   |
| INDEX (immediate, scalar)                     | `INDEX <Zd>.<T>, #<imm>, <R><m>`                                                    | ZINDEX            | 26   |
| INDEX (immediates)                            | `INDEX <Zd>.<T>, #<imm1>, #<imm2>`                                                  | ZINDEX            | 26   |
| INDEX (scalar, immediate)                     | `INDEX <Zd>.<T>, <R><n>, #<imm>`                                                    | ZINDEX            | 26   |
| INDEX (scalars)                               | `INDEX <Zd>.<T>, <R><n>, <R><m>`                                                    | ZINDEX            | 26   |
| LD1B (scalar plus immediate, single register) | `LD1B { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                           | ZLD1B             | 32   |
| LD1B (scalar plus scalar, single register)    | `LD1B { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>, <Xm>]`                                       | ZLD1B             | 33   |
| LD1D (scalar plus immediate, single register) | `LD1D { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                           | ZLD1D             | 32   |
| LD1D (scalar plus scalar, single register)    | `LD1D { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #3]`                               | ZLD1D             | 33   |
| LD1H (scalar plus immediate, single register) | `LD1H { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                           | ZLD1H             | 32   |
| LD1H (scalar plus scalar, single register)    | `LD1H { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #1]`                               | ZLD1H             | 33   |
| LD1SB (scalar plus immediate)                 | `LD1SB { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                          | ZLD1SB            | 32   |
| LD1SB (scalar plus scalar)                    | `LD1SB { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>, <Xm>]`                                      | ZLD1SB            | 33   |
| LD1SH (scalar plus immediate)                 | `LD1SH { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                          | ZLD1SH            | 32   |
| LD1SH (scalar plus scalar)                    | `LD1SH { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #1]`                              | ZLD1SH            | 33   |
| LD1SW (scalar plus immediate)                 | `LD1SW { <Zt>.D }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                            | ZLD1SW            | 32   |
| LD1SW (scalar plus scalar)                    | `LD1SW { <Zt>.D }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #2]`                                | ZLD1SW            | 33   |
| LD1W (scalar plus immediate, single register) | `LD1W { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                           | ZLD1W             | 32   |
| LD1W (scalar plus scalar, single register)    | `LD1W { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #2]`                               | ZLD1W             | 33   |
| LD2B (scalar plus immediate)                  | `LD2B { <Zt1>.B, <Zt2>.B }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                   | ZLD2B             | 34   |
| LD2B (scalar plus scalar)                     | `LD2B { <Zt1>.B, <Zt2>.B }, <Pg>/Z, [<Xn\|SP>, <Xm>]`                               | ZLD2B             | 35   |
| LD2D (scalar plus immediate)                  | `LD2D { <Zt1>.D, <Zt2>.D }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                   | ZLD2D             | 34   |
| LD2D (scalar plus scalar)                     | `LD2D { <Zt1>.D, <Zt2>.D }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #3]`                       | ZLD2D             | 35   |
| LD2H (scalar plus immediate)                  | `LD2H { <Zt1>.H, <Zt2>.H }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                   | ZLD2H             | 34   |
| LD2H (scalar plus scalar)                     | `LD2H { <Zt1>.H, <Zt2>.H }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #1]`                       | ZLD2H             | 35   |
| LD2Q (scalar plus immediate)                  | `LD2Q { <Zt1>.Q, <Zt2>.Q }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                   | ZLD2Q             | 34   |
| LD2Q (scalar plus scalar)                     | `LD2Q { <Zt1>.Q, <Zt2>.Q }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #4]`                       | ZLD2Q             | 35   |
| LD2W (scalar plus immediate)                  | `LD2W { <Zt1>.S, <Zt2>.S }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                   | ZLD2W             | 34   |
| LD2W (scalar plus scalar)                     | `LD2W { <Zt1>.S, <Zt2>.S }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #2]`                       | ZLD2W             | 35   |
| LD3B (scalar plus immediate)                  | `LD3B { <Zt1>.B, <Zt2>.B, <Zt3>.B }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`          | ZLD3B             | 34   |
| LD3B (scalar plus scalar)                     | `LD3B { <Zt1>.B, <Zt2>.B, <Zt3>.B }, <Pg>/Z, [<Xn\|SP>, <Xm>]`                      | ZLD3B             | 35   |
| LD3D (scalar plus immediate)                  | `LD3D { <Zt1>.D, <Zt2>.D, <Zt3>.D }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`          | ZLD3D             | 34   |
| LD3D (scalar plus scalar)                     | `LD3D { <Zt1>.D, <Zt2>.D, <Zt3>.D }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #3]`              | ZLD3D             | 35   |
| LD3H (scalar plus immediate)                  | `LD3H { <Zt1>.H, <Zt2>.H, <Zt3>.H }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`          | ZLD3H             | 34   |
| LD3H (scalar plus scalar)                     | `LD3H { <Zt1>.H, <Zt2>.H, <Zt3>.H }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #1]`              | ZLD3H             | 35   |
| LD3Q (scalar plus immediate)                  | `LD3Q { <Zt1>.Q, <Zt2>.Q, <Zt3>.Q }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`          | ZLD3Q             | 34   |
| LD3Q (scalar plus scalar)                     | `LD3Q { <Zt1>.Q, <Zt2>.Q, <Zt3>.Q }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #4]`              | ZLD3Q             | 35   |
| LD3W (scalar plus immediate)                  | `LD3W { <Zt1>.S, <Zt2>.S, <Zt3>.S }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`          | ZLD3W             | 34   |
| LD3W (scalar plus scalar)                     | `LD3W { <Zt1>.S, <Zt2>.S, <Zt3>.S }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #2]`              | ZLD3W             | 35   |
| LD4B (scalar plus immediate)                  | `LD4B { <Zt1>.B, <Zt2>.B, <Zt3>.B, <Zt4>.B }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]` | ZLD4B             | 34   |
| LD4B (scalar plus scalar)                     | `LD4B { <Zt1>.B, <Zt2>.B, <Zt3>.B, <Zt4>.B }, <Pg>/Z, [<Xn\|SP>, <Xm>]`             | ZLD4B             | 35   |
| LD4D (scalar plus immediate)                  | `LD4D { <Zt1>.D, <Zt2>.D, <Zt3>.D, <Zt4>.D }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]` | ZLD4D             | 34   |
| LD4D (scalar plus scalar)                     | `LD4D { <Zt1>.D, <Zt2>.D, <Zt3>.D, <Zt4>.D }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #3]`     | ZLD4D             | 35   |
| LD4H (scalar plus immediate)                  | `LD4H { <Zt1>.H, <Zt2>.H, <Zt3>.H, <Zt4>.H }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]` | ZLD4H             | 34   |
| LD4H (scalar plus scalar)                     | `LD4H { <Zt1>.H, <Zt2>.H, <Zt3>.H, <Zt4>.H }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #1]`     | ZLD4H             | 35   |
| LD4Q (scalar plus immediate)                  | `LD4Q { <Zt1>.Q, <Zt2>.Q, <Zt3>.Q, <Zt4>.Q }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]` | ZLD4Q             | 34   |
| LD4Q (scalar plus scalar)                     | `LD4Q { <Zt1>.Q, <Zt2>.Q, <Zt3>.Q, <Zt4>.Q }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #4]`     | ZLD4Q             | 35   |
| LD4W (scalar plus immediate)                  | `LD4W { <Zt1>.S, <Zt2>.S, <Zt3>.S, <Zt4>.S }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]` | ZLD4W             | 34   |
| LD4W (scalar plus scalar)                     | `LD4W { <Zt1>.S, <Zt2>.S, <Zt3>.S, <Zt4>.S }, <Pg>/Z, [<Xn\|SP>, <Xm>, LSL #2]`     | ZLD4W             | 35   |
| LDFF1B (scalar plus scalar)                   | `LDFF1B { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, <Xm>}]`                                   | ZLDFF1B           | 33   |
| LDFF1D (scalar plus scalar)                   | `LDFF1D { <Zt>.D }, <Pg>/Z, [<Xn\|SP>{, <Xm>, LSL #3}]`                             | ZLDFF1D           | 33   |
| LDFF1H (scalar plus scalar)                   | `LDFF1H { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, <Xm>, LSL #1}]`                           | ZLDFF1H           | 33   |
| LDFF1SB (scalar plus scalar)                  | `LDFF1SB { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, <Xm>}]`                                  | ZLDFF1SB          | 33   |
| LDFF1SH (scalar plus scalar)                  | `LDFF1SH { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, <Xm>, LSL #1}]`                          | ZLDFF1SH          | 33   |
| LDFF1SW (scalar plus scalar)                  | `LDFF1SW { <Zt>.D }, <Pg>/Z, [<Xn\|SP>{, <Xm>, LSL #2}]`                            | ZLDFF1SW          | 33   |
| LDFF1W (scalar plus scalar)                   | `LDFF1W { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, <Xm>, LSL #2}]`                           | ZLDFF1W           | 33   |
| LDNF1B                                        | `LDNF1B { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                         | ZLDNF1B           | 32   |
| LDNF1D                                        | `LDNF1D { <Zt>.D }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                           | ZLDNF1D           | 32   |
| LDNF1H                                        | `LDNF1H { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                         | ZLDNF1H           | 32   |
| LDNF1SB                                       | `LDNF1SB { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                        | ZLDNF1SB          | 32   |
| LDNF1SH                                       | `LDNF1SH { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                        | ZLDNF1SH          | 32   |
| LDNF1SW                                       | `LDNF1SW { <Zt>.D }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                          | ZLDNF1SW          | 32   |
| LDNF1W                                        | `LDNF1W { <Zt>.<T> }, <Pg>/Z, [<Xn\|SP>{, #<imm>, MUL VL}]`                         | ZLDNF1W           | 32   |
| LSL (immediate, predicated)                   | `LSL <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`                                        | ZLSL              | 15   |
| LSL (immediate, unpredicated)                 | `LSL <Zd>.<T>, <Zn>.<T>, #<const>`                                                  | ZLSL              | 18   |
| LSL (vectors)                                 | `LSL <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZLSL              | 16   |
| LSL (wide elements, predicated)               | `LSL <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.D`                                          | ZLSL              | 16   |
| LSL (wide elements, unpredicated)             | `LSL <Zd>.<T>, <Zn>.<T>, <Zm>.D`                                                    | ZLSL              | 19   |
| LSLR                                          | `LSLR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZLSLR             | 16   |
| LSR (immediate, predicated)                   | `LSR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`                                        | ZLSR              | 15   |
| LSR (immediate, unpredicated)                 | `LSR <Zd>.<T>, <Zn>.<T>, #<const>`                                                  | ZLSR              | 18   |
| LSR (vectors)                                 | `LSR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZLSR              | 16   |
| LSR (wide elements, predicated)               | `LSR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.D`                                          | ZLSR              | 16   |
| LSR (wide elements, unpredicated)             | `LSR <Zd>.<T>, <Zn>.<T>, <Zm>.D`                                                    | ZLSR              | 19   |
| LSRR                                          | `LSRR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZLSRR             | 16   |
| MUL (immediate)                               | `MUL <Zdn>.<T>, <Zdn>.<T>, #<imm>`                                                  | ZMUL              | 3    |
| MUL (indexed)                                 | `MUL <Zd>.<T>, <Zn>.<T>, <Zm>.<T>[<imm>]`                                           | ZMUL              | 9    |
| MUL (vectors, predicated)                     | `MUL <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZMUL              | 2    |
| MUL (vectors, unpredicated)                   | `MUL <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                  | ZMUL              | 1    |
| NAND                                          | `NAND <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                               | PNAND             | 6    |
| NANDS                                         | `NANDS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                              | PNANDS            | 6    |
| NEG                                           | `NEG <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                    | ZNEG              | 17   |
| NOR                                           | `NOR <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                                | PNOR              | 6    |
| NORS                                          | `NORS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                               | PNORS             | 6    |
| NOT (vector)                                  | `NOT <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                    | ZNOT              | 17   |
| ORN (immediate)                               | `ORN <Zdn>.<T>, <Zdn>.<T>, #<const>`                                                | ZORN              | 5    |
| ORN (predicates)                              | `ORN <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                                | PORN              | 6    |
| ORNS                                          | `ORNS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                               | PORNS             | 6    |
| ORR (immediate)                               | `ORR <Zdn>.<T>, <Zdn>.<T>, #<const>`                                                | ZORR              | 5    |
| ORR (predicates)                              | `ORR <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                                | ZORR              | 6    |
| ORR (vectors, predicated)                     | `ORR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZORR              | 2    |
| ORR (vectors, unpredicated)                   | `ORR <Zd>.D, <Zn>.D, <Zm>.D`                                                        | ZORR              | 4    |
| ORRS                                          | `ORRS <Pd>.B, <Pg>/Z, <Pn>.B, <Pm>.B`                                               | PORRS             | 6    |
| PFALSE                                        | `PFALSE <Pd>.B`                                                                     | PFALSE            | 22   |
| PFIRST                                        | `PFIRST <Pdn>.B, <Pg>, <Pdn>.B`                                                     | PFIRST            | 21   |
| PMUL                                          | `PMUL <Zd>.B, <Zn>.B, <Zm>.B`                                                       | ZPMUL             | 1    |
| PTEST                                         | `PTEST <Pg>, <Pn>.B`                                                                | PTEST             | 20   |
| PTRUE (predicate)                             | `PTRUE <Pd>.<T>{, <pattern>}`                                                       | PTRUE             | 25   |
| PTRUES                                        | `PTRUES <Pd>.<T>{, <pattern>}`                                                      | PTRUES            | 25   |
| PNEXT                                         | `PNEXT <Pdn>.<T>, <Pv>, <Pdn>.<T>`                                                  | PNEXT             | 24   |
| PUNPKHI                                       | `PUNPKHI <Pd>.H, <Pn>.B`                                                            | PUNPKHI           | 27   |
| PUNPKLO                                       | `PUNPKLO <Pd>.H, <Pn>.B`                                                            | PUNPKLO           | 27   |
| RDFFR (predicated)                            | `RDFFR <Pd>.B, <Pg>/Z`                                                              | PRDFFR            | 23   |
| RDFFR (unpredicated)                          | `RDFFR <Pd>.B`                                                                      | PRDFFR            | 23   |
| RDFFRS                                        | `RDFFRS <Pd>.B, <Pg>/Z`                                                             | PRDFFRS           | 23   |
| SABD                                          | `SABD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZSABD             | 2    |
| SDIV                                          | `SDIV <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZSDIV             | 2    |
| SDIVR                                         | `SDIVR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                      | ZSDIVR            | 2    |
| SEL (predicates)                              | `SEL <Pd>.B, <Pg>, <Pn>.B, <Pm>.B`                                                  | PSEL              | 6    |
| SEL (vectors)                                 | `SEL <Zd>.<T>, <Pv>, <Zn>.<T>, <Zm>.<T>`                                            | ZSEL              | 14   |
| SMAX (immediate)                              | `SMAX <Zdn>.<T>, <Zdn>.<T>, #<imm>`                                                 | ZSMAX             | 3    |
| SMAX (vectors)                                | `SMAX <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZSMAX             | 2    |
| SMIN (immediate)                              | `SMIN <Zdn>.<T>, <Zdn>.<T>, #<imm>`                                                 | ZSMIN             | 3    |
| SMIN (vectors)                                | `SMIN <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZSMIN             | 2    |
| SMULH (predicated)                            | `SMULH <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                      | ZSMULH            | 2    |
| SMULH (unpredicated)                          | `SMULH <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                | ZSMULH            | 1    |
| SQADD (immediate)                             | `SQADD <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`                                     | ZSQADD            | 3    |
| SQADD (vectors, predicated)                   | `SQADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                      | ZSQADD            | 2    |
| SQADD (vectors, unpredicated)                 | `SQADD <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                | ZSQADD            | 1    |
| SQDMULH (vectors)                             | `SQDMULH <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                              | ZSQDMULH          | 1    |
| SQRDMULH (vectors)                            | `SQRDMULH <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                             | ZSQRDMULH         | 1    |
| SQSHL (immediate)                             | `SQSHL <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`                                      | ZSQSHL            | 15   |
| SQSHLU                                        | `SQSHLU <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`                                     | ZSQSHLU           | 15   |
| SQSUB (immediate)                             | `SQSUB <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`                                     | ZSQSUB            | 3    |
| SQSUB (vectors, predicated)                   | `SQSUB <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                      | ZSQSUB            | 2    |
| SQSUB (vectors, unpredicated)                 | `SQSUB <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                | ZSQSUB            | 1    |
| SQSUBR                                        | `SQSUBR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                     | ZSQSUBR           | 2    |
| SRSHR                                         | `SRSHR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`                                      | ZSRSHR            | 15   |
| SUB (immediate)                               | `SUB <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`                                       | ZSUB              | 3    |
| SUB (vectors, predicated)                     | `SUB <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                        | ZSUB              | 2    |
| SUB (vectors, unpredicated)                   | `SUB <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                  | ZSUB              | 1    |
| SUBR (immediate)                              | `SUBR <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`                                      | ZSUBR             | 3    |
| SUBR (vectors)                                | `SUBR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZSUBR             | 2    |
| SUQADD                                        | `SUQADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                     | ZSUQADD           | 2    |
| SXTB                                          | `SXTB <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                   | ZSXTB             | 17   |
| SXTH                                          | `SXTH <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                   | ZSXTH             | 17   |
| SXTW                                          | `SXTW <Zd>.D, <Pg>/M, <Zn>.D`                                                       | ZSXTW             | 17   |
| TRN1 (predicates)                             | `TRN1 <Pd>.<T>, <Pn>.<T>, <Pm>.<T>`                                                 | UTRN1             | 28   |
| TRN1 (vectors)                                | `TRN1 <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                 | ZTRN1             | 29   |
| TRN2 (predicates)                             | `TRN2 <Pd>.<T>, <Pn>.<T>, <Pm>.<T>`                                                 | UTRN2             | 28   |
| TRN2 (vectors)                                | `TRN2 <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                 | ZTRN2             | 29   |
| UABD                                          | `UABD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZUABD             | 2    |
| UDIV                                          | `UDIV <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZUDIV             | 2    |
| UDIVR                                         | `UDIVR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                      | ZUDIVR            | 2    |
| UMAX (immediate)                              | `UMAX <Zdn>.<T>, <Zdn>.<T>, #<imm>`                                                 | ZUMAX             | 3    |
| UMAX (vectors)                                | `UMAX <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZUMAX             | 2    |
| UMIN (immediate)                              | `UMIN <Zdn>.<T>, <Zdn>.<T>, #<imm>`                                                 | ZUMIN             | 3    |
| UMIN (vectors)                                | `UMIN <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                       | ZUMIN             | 2    |
| UMULH (predicated)                            | `UMULH <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                      | ZUMULH            | 2    |
| UMULH (unpredicated)                          | `UMULH <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                | ZUMULH            | 1    |
| UQADD (immediate)                             | `UQADD <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`                                     | ZUQADD            | 3    |
| UQADD (vectors, predicated)                   | `UQADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                      | ZUQADD            | 2    |
| UQADD (vectors, unpredicated)                 | `UQADD <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                | ZUQADD            | 1    |
| UQSHL (immediate)                             | `UQSHL <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`                                      | ZUQSHL            | 15   |
| UQSUB (immediate)                             | `UQSUB <Zdn>.<T>, <Zdn>.<T>, #<imm>{, <shift>}`                                     | ZUQSUB            | 3    |
| UQSUB (vectors, predicated)                   | `UQSUB <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                      | ZUQSUB            | 2    |
| UQSUB (vectors, unpredicated)                 | `UQSUB <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                | ZUQSUB            | 1    |
| UQSUBR                                        | `UQSUBR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                     | ZUQSUBR           | 2    |
| URSHR                                         | `URSHR <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, #<const>`                                      | ZURSHR            | 15   |
| USQADD                                        | `USQADD <Zdn>.<T>, <Pg>/M, <Zdn>.<T>, <Zm>.<T>`                                     | ZUSQADD           | 2    |
| UXTB                                          | `UXTB <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                   | ZUXTB             | 17   |
| UXTH                                          | `UXTH <Zd>.<T>, <Pg>/M, <Zn>.<T>`                                                   | ZUXTH             | 17   |
| UXTW                                          | `UXTW <Zd>.D, <Pg>/M, <Zn>.D`                                                       | ZUXTW             | 17   |
| UZP1 (predicates)                             | `UZP1 <Pd>.<T>, <Pn>.<T>, <Pm>.<T>`                                                 | PUZP1             | 28   |
| UZP1 (vectors)                                | `UZP1 <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                 | ZUZP1             | 29   |
| UZP2 (predicates)                             | `UZP2 <Pd>.<T>, <Pn>.<T>, <Pm>.<T>`                                                 | PUZP2             | 28   |
| UZP2 (vectors)                                | `UZP2 <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                 | ZUZP2             | 29   |
| WHILEGE (predicate pair)                      | `WHILEGE { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`                                      | WHILEGE           | 11   |
| WHILEGE (predicate)                           | `WHILEGE <Pd>.<T>, <R><n>, <R><m>`                                                  | WHILEGE, WHILEGEW | 10   |
| WHILEGT (predicate pair)                      | `WHILEGT { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`                                      | WHILEGT           | 11   |
| WHILEGT (predicate)                           | `WHILEGT <Pd>.<T>, <R><n>, <R><m>`                                                  | WHILEGT, WHILEGTW | 10   |
| WHILEHI (predicate pair)                      | `WHILEHI { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`                                      | WHILEHI           | 11   |
| WHILEHI (predicate)                           | `WHILEHI <Pd>.<T>, <R><n>, <R><m>`                                                  | WHILEHI, WHILEHIW | 10   |
| WHILEHS (predicate pair)                      | `WHILEHS { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`                                      | WHILEHS           | 11   |
| WHILEHS (predicate)                           | `WHILEHS <Pd>.<T>, <R><n>, <R><m>`                                                  | WHILEHS, WHILEHSW | 10   |
| WHILELE (predicate pair)                      | `WHILELE { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`                                      | WHILELE           | 11   |
| WHILELE (predicate)                           | `WHILELE <Pd>.<T>, <R><n>, <R><m>`                                                  | WHILELE, WHILELEW | 10   |
| WHILELO (predicate pair)                      | `WHILELO { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`                                      | WHILELO           | 11   |
| WHILELO (predicate)                           | `WHILELO <Pd>.<T>, <R><n>, <R><m>`                                                  | WHILELO, WHILELOW | 10   |
| WHILELS (predicate pair)                      | `WHILELS { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`                                      | WHILELS           | 11   |
| WHILELS (predicate)                           | `WHILELS <Pd>.<T>, <R><n>, <R><m>`                                                  | WHILELS, WHILELSW | 10   |
| WHILELT (predicate pair)                      | `WHILELT { <Pd1>.<T>, <Pd2>.<T> }, <Xn>, <Xm>`                                      | WHILELT           | 11   |
| WHILELT (predicate)                           | `WHILELT <Pd>.<T>, <R><n>, <R><m>`                                                  | WHILELT, WHILELTW | 10   |
| ZIP1 (predicates)                             | `ZIP1 <Pd>.<T>, <Pn>.<T>, <Pm>.<T>`                                                 | PZIP1             | 28   |
| ZIP1 (vectors)                                | `ZIP1 <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                 | ZZIP1             | 29   |
| ZIP2 (predicates)                             | `ZIP2 <Pd>.<T>, <Pn>.<T>, <Pm>.<T>`                                                 | PZIP2             | 28   |
| ZIP2 (vectors)                                | `ZIP2 <Zd>.<T>, <Zn>.<T>, <Zm>.<T>`                                                 | ZZIP2             | 29   |
