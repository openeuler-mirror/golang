# ARM64 SVE 汇编器支持测试设计文档

## 1. 测试背景

本测试设计针对 Go ARM64 汇编器新增的 SVE（Scalable Vector Extension）支持，围绕以下能力设计测试：

- ARM64 SVE `Z` 向量寄存器和 `P` 谓词寄存器解析。
- `Zn.<T>`、`Pn.<T>`、`Pg/M`、`Pg/Z` 等 SVE 复合操作数解析。
- SVE pattern specifier、VL 缩放地址、SVE register list 等语法解析。
- SVE 指令 `optab` 匹配和 `asmoutsve` 编码分发。
- SVE 相关寄存器和操作数 pretty print。
- 原 ARM64 操作数分类 `C_ZREG` 重命名为 `C_REGZR`，避免与 SVE `Z` 寄存器冲突。

测试目标是验证新增 SVE 支持正确、稳定、可诊断。既有 ARM64 汇编功能回归由已有测试套承担，不纳入本次测试设计范围。

## 2. 测试目标

本轮测试需要验证：

- Go 汇编器能正确解析已实现的 SVE 汇编语法。
- SVE 操作数能被正确分类，并匹配到预期 `optab` 项。
- SVE 指令能生成正确机器码。
- 非法 SVE 输入能被拒绝，不能静默生成错误编码。
- 对部分指令中只能编码 `P0` 到 `P7` 的谓词寄存器约束进行边界验证。
- pretty print / 反汇编输出可读且语义正确。
- 在 SVE 相关场景中验证 `Z`/`P` 寄存器不会与 `ZR` 等既有 ARM64 寄存器语义混淆。

## 3. 测试范围

### 3.1 范围内

- `cmd/asm` 的 ARM64 SVE 语法解析。
- `cmd/internal/obj/arm64` 的寄存器表示、操作数分类、指令匹配和机器码编码。
- `sve.md` 中已声明支持的指令和 case。
- SVE 操作数 pretty print。
- `operand_test.go`、`arm64enc.s`、`arm64error.s` 中的 SVE 相关测试数据。

### 3.2 范围外

- SVE 指令在真实硬件上的运行时语义验证。
- Go 编译器自动生成 SVE 指令能力。
- 未列入 `sve.md` 的未来 SVE 指令设计。
- 性能优化或 benchmark。
- ARM64 既有汇编功能回归。该部分由已有测试套覆盖。
- Loong64、MIPS、S390X、RISCV、Wasm 等跨架构回归测试。

## 4. 测试策略

本轮测试设计包含三类仓内自动化测试和一类补充人工检查：

- 操作数解析测试：在 `operand_test.go` 中覆盖 SVE 寄存器、arrangement、predication、元素索引和 VL 地址的解析/打印结果。
- 编码 golden 测试：在 `arm64enc.s` 中覆盖 `sve.md` 已声明的 SVE 助记符，每个助记符至少有一条可编码样例。
- 负向 golden 测试：在 `arm64error.s` 中覆盖多类 governing predicate `P0-P7` 约束，防止 `P8` 被低位截断误编码。
- objdump 补充测试：对正向编码对象进行反汇编抽查，确认 golden 编码的反汇编语义与测试输入一致。

本轮不穷举同一助记符的所有 addressing mode、所有数据宽度、所有立即数边界和所有非法组合。

## 5. 操作数解析测试设计

操作数解析测试落在 `src/cmd/asm/internal/asm/operand_test.go` 的 `TestARM64OperandParser`。

### 5.1 正向解析用例

| 类别 | 用例 | 验证点 |
|---|---|---|
| SVE 向量寄存器 | `Z0`、`Z31` | `Z` 寄存器边界解析，不与 `ZR` 混淆 |
| SVE 谓词寄存器 | `P0`、`P15` | `P` 寄存器边界解析 |
| 向量 arrangement | `Z0.B`、`Z1.H`、`Z2.S`、`Z3.D`、`Z4.Q` | `Z` 寄存器宽度后缀解析 |
| 谓词 arrangement | `P0.B`、`P1.H`、`P2.S`、`P3.D` | `P` 寄存器宽度后缀解析 |
| predication | `P0/M`、`P7/M`、`P0/Z`、`P7/Z` | merge/zero predication 解析，并覆盖 governing predicate 合法边界 `P7` |
| 元素索引 | `Z0.B[0]` | SVE vector element 解析 |
| VL 地址 | `(VL*1)(R0)`、`(VL*-1)(RSP)` | 正/负 VL 缩放地址和 SP 基址解析 |

### 5.2 负向解析用例

| 用例 | 预期错误 | 验证点 |
|---|---|---|
| `P0/X` | `invalid predication` | 非法 predication 后缀被拒绝 |
| `Z0/M` | `invalid governing predicate registers` | 非谓词寄存器不能使用 governing predicate 后缀 |

## 6. 编码 golden 测试设计

编码 golden 测试落在 `src/cmd/asm/internal/asm/testdata/arm64enc.s`，由 `TestARM64Encoder` 执行。

### 6.1 助记符覆盖

正向编码用例按 `sve.md` 中已声明的 SVE 助记符进行设计：

- `sve.md` 中已声明的唯一 SVE 助记符应在 `arm64enc.s` 中至少出现一次。
- 每条正向用例包含预期机器码 golden，用于校验 Go 汇编器编码结果。
- 用例选择以覆盖助记符和主要编码模板为目标，不对每个助记符的所有变体做穷举。

### 6.2 重点边界用例

| 类别 | 用例方向 | 验证点 |
|---|---|---|
| predicated vector | `ZADD ... P0/M ...`、`ZADD ... P7/M ...` | governing predicate 合法边界 `P0`/`P7` |
| SVE load | `ZLD1B ... P0/Z ...`、`ZLD1B ... P7/Z ...` | load 中 governing predicate 合法边界 |
| SVE store | `ZST1B ... P0 ...`、`ZST1B ... P7 ...` | store 中 governing predicate 合法边界 |
| pattern | `PTRUE P0.B`、`PTRUE VL16, P1.S` | 默认 pattern 和显式 pattern 编码 |
| VL 地址 | `ZLD* (VL*n)(R0)`、`ZST* ... (VL*n)(R0)` | VL 缩放地址编码 |
| register list | `ZLD2*`/`ZLD3*`/`ZLD4*`、`ZST2*`/`ZST3*`/`ZST4*` | 多寄存器 list 编码 |
| predicate 类指令 | `PAND`、`PBRK*`、`PZIP*` 等 | 谓词寄存器字段和 arrangement 编码 |
| scalar/SIMD 目的 | `ZLASTA`、`ZLASTB`、`ZSADDV`、`ZUADDV` | SVE 到通用/SIMD 寄存器目的编码 |

## 7. 负向测试设计

负向测试分为解析层负向和编码层负向。

### 7.1 解析层负向

解析层负向用例位于 `operand_test.go`：

- `P0/X`：非法 predication 后缀。
- `Z0/M`：非谓词寄存器使用 predication 后缀。

### 7.2 编码层负向

编码层负向用例位于 `src/cmd/asm/internal/asm/testdata/arm64error.s`，由 `TestARM64Errors` 执行。

| 用例方向 | 预期错误 |
|---|---|
| `ZADD ... P8/M ...` | `invalid governing scalable predicate register P0-P7` |
| `ZCMPEQ ... P8/Z ...` | `invalid governing scalable predicate register P0-P7` |
| `ZASR ... P8/M ...` | `invalid governing scalable predicate register P0-P7` |
| `ZABS ... P8/M ...` | `invalid governing scalable predicate register P0-P7` |
| `ZLD1B ... P8/Z ...` | `invalid governing scalable predicate register P0-P7` |
| `ZST1B ... P8 ...` | `invalid governing scalable predicate register P0-P7` |
| `ZSADDV ... P8 ...` | `invalid governing scalable predicate register P0-P7` |

这些用例覆盖 predicated vector、compare、shift immediate、unary、SVE load、SVE store、reduction 等路径，确认 `P8` 不能被低位截断为 `P0` 后继续编码。

## 8. 打印与反汇编测试设计

打印相关检查分为两部分：

- `operand_test.go` 通过输入/期望字符串验证 SVE 操作数的规范化输出，例如 `Z0.B`、`P0/M`、`Z0.B[0]`、`(VL*1)(R0)`。
- 第 11 章的 objdump 补充测试通过反汇编对象文件，人工抽查正向编码用例的反汇编语义。

本轮不要求新增独立反汇编 golden 文件，也不要求对所有 SVE 指令逐条归档 objdump 结果。

## 9. SVE 特性边界测试设计

本轮重点验证以下 SVE 特性边界：

- `Z0`/`Z31` 与 `ZR` 的语义边界。
- `P0`/`P15` 的谓词寄存器解析边界。
- `P7` 与 `P8` 在 governing predicate 场景中的合法/非法边界。
- VL 缩放地址和 SVE register list 的编码路径。

ARM64 既有汇编功能回归由已有测试套覆盖；跨架构测试不在本轮范围内。

## 10. 测试数据放置

| 测试类型 | 文件 | 执行测试 |
|---|---|---|
| 操作数正向解析 | `src/cmd/asm/internal/asm/operand_test.go` | `TestARM64OperandParser` |
| 操作数负向解析 | `src/cmd/asm/internal/asm/operand_test.go` | `TestARM64OperandParser` |
| 正向编码 golden | `src/cmd/asm/internal/asm/testdata/arm64enc.s` | `TestARM64Encoder` |
| 负向编码 golden | `src/cmd/asm/internal/asm/testdata/arm64error.s` | `TestARM64Errors` |
| objdump 补充测试 | 临时对象文件 `/tmp/arm64-sve-asm.o` | 手工抽查 |

## 11. 测试执行

### 11.1 主验证命令

```bash
export GOROOT=$(pwd)
export PATH=$GOROOT/bin:$PATH
cd src
go test ./cmd/asm/internal/asm ./cmd/asm/...
```

通过标准：

```text
ok  	cmd/asm/internal/asm	...
ok  	cmd/asm/internal/lex	...
```

`cmd/asm`、`cmd/asm/internal/arch`、`cmd/asm/internal/flags` 显示 `[no test files]` 属于正常结果。

### 11.2 objdump 补充测试

该项用于补充确认正向编码用例的反汇编语义，不替代主验证命令。

```bash
export GOROOT=$(pwd)
export PATH=$GOROOT/bin:$PATH
cd src
go tool asm -I pkg/include -o /tmp/arm64-sve-asm.o cmd/asm/internal/asm/testdata/arm64enc.s
objdump -d /tmp/arm64-sve-asm.o
```

重点核对：

- 反汇编出的 SVE 指令助记符与测试输入一致。
- 操作数顺序与 Go 汇编语法预期一致。
- 数据宽度、谓词方式、VL 地址和 register list 展示正确。
- `P0-P7` governing predicate 边界相关用例没有出现低位截断导致的误编码。

如存在差异，记录差异指令、测试输入和反汇编输出。

## 12. 验收标准

本特性测试通过需满足：

- 主验证命令执行通过。
- 操作数解析正向和负向用例结果符合预期。
- `arm64enc.s` 中 SVE 正向编码 golden 全部通过。
- `arm64error.s` 中各类 `P8` governing predicate 负向用例全部失败并输出预期错误。
- `sve.md` 中已声明的唯一 SVE 助记符均至少有一条正向编码用例。
- `PTRUE` 默认 pattern、显式 pattern，SVE VL 地址和多寄存器 list 均有代表用例。
- objdump 补充测试用于辅助验收；执行时应记录反汇编抽查结果，存在差异时记录差异指令、测试输入和反汇编输出。
- ARM64 既有汇编功能回归不在本轮验收范围内，由已有测试套承担。
- 跨架构测试不在本轮验收范围内，相关影响作为残余风险记录。

## 13. 残余风险

本轮测试完成后仍保留以下风险：

- 同一助记符的所有 addressing mode、数据宽度、立即数边界未穷举。
- 非法组合仅覆盖高风险样例，不覆盖所有错误输入。
- objdump 为补充抽查，本轮不要求新增独立反汇编 golden 文件。
- SVE 指令运行时语义不在本轮验证范围内。
- `C_REGZR` 相关既有 ARM64 行为由已有测试套覆盖，本轮仅关注 SVE 场景内的语义边界。
