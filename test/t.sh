#!/bin/bash
export TMPDIR=${PWD}/tmp
export TEST_GCFLAGS='-d=go119usejumptables=0'
export TEST_LDFLAGS='-linkmode=external -extld=gcc -extldflags "-no-pie -Wl,--compress-debug-sections=none -Wl,-q"'
export TEST_BUILDFLAGS="-buildmode=pie -tags=goboltci,bolt"
if [[ $(uname -i) == aarch64 ]]; then
    export TEST_BUILDFLAGS="${TEST_BUILDFLAGS} -mappingsymbol"
fi
export GO_TEST_SCRIPT=${PWD}/bolt.sh
 
export RUN_SUBREAPER=${PWD}/run_subreaper
 
export GO_TEST_MODE="instr_test"
export LOGSFILE=${PWD}/logs.log
 
rm -rf $LOGSFILE
rm -rf ${TMPDIR}
mkdir -p ${TMPDIR}

if [[ ! -d ${BOLT_DIR} ]]; then
	echo "ERROR: BOLT_DIR env var should point to existing directory with llvm-bolt. Exiting."
	exit 1
fi

if [[ ! -x ${PWD}/run_subreaper ]]; then
	gcc ${PWD}/run_subreaper.c -o ${PWD}/run_subreaper
fi

${PWD}/goboltrm.bash

set -x

${TASKSET} ../bin/go clean -cache
${TASKSET} ../bin/go test -a -tags=goboltci,bolt cmd/internal/testdir -k -test.timeout=30m -test.v

