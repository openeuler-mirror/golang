#!/bin/bash
 
# Parse args
exe=$1
exe_path=$(dirname $exe)
shift
args=$@
 
# Set vars
ret=0
test_var=""
RUN_SUBREAPER=${RUN_SUBREAPER:-""}
PERF=${PERF:-perf}
LOGSFILE=${LOGSFILE:-/tmp/test.log}
DEVNULL=${DEVNULL:-/dev/null}
MAX_SLEEP_TIME=${MAX_SLEEP_TIME:-30}
 
BOLT_GOLANG="-golang=1"
BOLT_FLAGS="-reorder-blocks=cache+ -reorder-functions=hfsort+ --keep-nops"
BOLT_INSTR_FLAGS="-instrument -conservative-instrumentation -instrumentation-file=$exe_path/prof.fdata -instrumentation-sleep-time=$MAX_SLEEP_TIME -instrumentation-no-counters-clear -instrumentation-wait-forks=1"
 
# Extra flags
PERF_FLAGS="-b"
PERF2BOLT_FLAGS=""
if [ "$(uname -p)" == "aarch64" ]; then
    # Disable LBR for aarch64
    PERF_FLAGS=""
    PERF2BOLT_FLAGS="-nl"
fi
 
get_test_var() {
    local name=$1
    local tmp="$name$(basename $exe_path)"
    eval "test_var=\${$tmp}"
}
 
bad_exit() {
    local ret=$1
    local msg=$2
    get_test_var "filename"
    printf "%-10s\t %-80s\t %s\n" "$exe" "$test_var" "$msg" >> $LOGSFILE
    sync $LOGSFILE
    exit $ret
}
 
check_ret() {
    local ret=$1
    local msg=$2
    if [ $ret -ne 0 ]; then
        bad_exit "$ret" "$msg"
    fi
}
 
check_exp_ret() {
    local ret=$1
    local msg=$2
    local log=$3
 
    get_test_var "expect"
    if ([ $ret -ne 0 ] && [ "$test_var" != "true" ]) || ([ $ret -eq 0 ] && [ "$test_var" == "true" ]); then
        bad_exit "$ret" "$msg"
    fi
 
    get_test_var "outputIsSet"
    if [ "$test_var" == "true" ]; then
        get_test_var "output"
        diff $log <(echo -n "$test_var") &> $exe_path/diff.log
        check_ret $? "Wrong output for $msg"
    fi
}
 
check_file() {
    local file=$1
    if [ ! -f $file ]; then
        return -1
    fi
 
    fuser -s $file &> $DEVNULL
    if [ $? -eq 0 ]; then
        return -1
    fi
 
    return 0
}
 
check_prof_file() {
    local file=$1
    local max_sleep=$MAX_SLEEP_TIME
 
    check_file $file
    local ret=$?
    while [ $ret -ne 0 ] && [ $max_sleep -ne 0 ]; do
        sleep 1
        max_sleep=$((max_sleep - 1))
        check_file $file
        ret=$?
    done
 
    if [ $max_sleep -eq 0 ]; then
       bad_exit -1 "Failed to get prof file"
    fi
 
    sync $file
}
 
# Check original file
$exe $args &> $exe_path/orig.log
ret=$?
check_exp_ret $ret "Original exec run" $exe_path/orig.log
case $GO_TEST_MODE in
    "run_only")
        cat $exe_path/orig.log
        ;;
 
    "perf_test")
        $PERF record -N -e cycles:u $PERF_FLAGS -F max -o $exe_path/perf.data -- $exe $args &> $exe_path/perf.log
        check_ret $? "Perf record"
        check_prof_file $exe_path/perf.data
        $BOLT_DIR/perf2bolt $PERF2BOLT_FLAGS -p $exe_path/perf.data -o $exe_path/perf.fdata $exe &> $exe_path/perf2bolt.log
        check_ret $? "perf2bolt"
        $BOLT_DIR/llvm-bolt $exe -o $exe.perf.bolt -data $exe_path/perf.fdata $BOLT_GOLANG $BOLT_FLAGS &> $exe_path/bolt.log
        check_ret $? "llvm-bolt"
        $exe.perf.bolt $args |& tee $exe_path/bolted.log
        ret=${PIPESTATUS[0]}
        check_exp_ret $ret "Bolted exec run" $exe_path/bolted.log
        ;;
 
    "instr_test")
        $BOLT_DIR/llvm-bolt $exe -o $exe.instr.tmp $BOLT_GOLANG $BOLT_INSTR_FLAGS &> $exe_path/bolt_instr.log
        check_ret $? "llvm-bolt instr"
        $RUN_SUBREAPER $exe.instr.tmp $args &> $exe_path/bolted_tmp.log
        check_ret $? "Instr exec run"
        check_prof_file $exe_path/prof.fdata
        $BOLT_DIR/llvm-bolt $exe -o $exe.instr.bolt -data $exe_path/prof.fdata $BOLT_GOLANG $BOLT_FLAGS &> $exe_path/bolt.log
        check_ret $? "llvm-bolt"
        $exe.instr.bolt $args |& tee $exe_path/bolted.log
        ret=${PIPESTATUS[0]}
        check_exp_ret $ret "Bolted exec run" $exe_path/bolted.log
        ;;
 
    *)
        echo "Bad GO_TEST_MODE"
        exit 1
        ;;
esac
 
sync
exit $ret
