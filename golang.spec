%global debug_package %{nil}
%global _binaries_in_noarch_packages_terminate_build 0
%global golibdir %{_libdir}/golang
%global goroot /usr/lib/%{name}
%global go_api 1.15
%global go_version 1.15
%global __spec_install_post /usr/lib/rpm/check-rpaths /usr/lib/rpm/check-buildroot /usr/lib/rpm/brp-compress
%global __requires_exclude_from ^(%{_datadir}|/usr/lib)/%{name}/(doc|src)/.*$
%global __strip /bin/true
%define _use_internal_dependency_generator 0
%define __find_requires %{nil}

%bcond_with bootstrap
%ifarch x86_64 aarch64
%bcond_without ignore_tests
%else
%bcond_with ignore_tests
%endif

%ifarch x86_64 aarch64
%global external_linker 1
%else
%global external_linker 0
%endif

%ifarch x86_64 aarch64
%global cgo_enabled 1
%else
%global cgo_enabled 0
%endif

%if %{with bootstrap}
%global golang_bootstrap 0
%else
%global golang_bootstrap 1
%endif

%if %{with ignore_tests}
%global fail_on_tests 0
%else
%global fail_on_tests 1
%endif

%global shared 0

%ifarch x86_64
%global race 1
%else
%global race 0
%endif

%ifarch x86_64
%global gohostarch amd64
%endif
%ifarch aarch64
%global gohostarch arm64
%endif

Name:           golang
Version:        1.15.7
Release:        25
Summary:        The Go Programming Language
License:        BSD and Public Domain
URL:            https://golang.org/
Source0:        https://dl.google.com/go/go%{version}.src.tar.gz

%if !%{golang_bootstrap}
BuildRequires:  gcc-go >= 5
%else
BuildRequires:  golang > 1.4
%endif
BuildRequires:  hostname
# for tests
BuildRequires:  pcre-devel, glibc-static, perl-interpreter, procps-ng

Provides:       go = %{version}-%{release}
Requires:       %{name}-devel = %{version}-%{release}

Obsoletes:      %{name}-pkg-bin-linux-386 < 1.4.99
Obsoletes:      %{name}-pkg-bin-linux-amd64 < 1.4.99
Obsoletes:      %{name}-pkg-bin-linux-arm < 1.4.99
Obsoletes:      %{name}-pkg-linux-386 < 1.4.99
Obsoletes:      %{name}-pkg-linux-amd64 < 1.4.99
Obsoletes:      %{name}-pkg-linux-arm < 1.4.99
Obsoletes:      %{name}-vet < 0-12.1
Obsoletes:      %{name}-cover < 0-12.1

Requires(post): %{_sbindir}/update-alternatives
Requires(postun): %{_sbindir}/update-alternatives
Requires:       glibc gcc git subversion


# generated by:
# go list -f {{.ImportPath}} ./src/vendor/... | sed "s:_$PWD/src/vendor/::g;s:_:.:;s:.*:Provides\: bundled(golang(&)):" && go list -f {{.ImportPath}} ./src/cmd/vendor/... | sed "s:_$PWD/src/cmd/vendor/::g;s:_:.:;s:.*:Provides\: bundled(golang(&)):"
Provides: bundled(golang(golang.org/x/crypto/chacha20poly1305))
Provides: bundled(golang(golang.org/x/crypto/cryptobyte))
Provides: bundled(golang(golang.org/x/crypto/cryptobyte/asn1))
Provides: bundled(golang(golang.org/x/crypto/curve25519))
Provides: bundled(golang(golang.org/x/crypto/internal/chacha20))
Provides: bundled(golang(golang.org/x/crypto/poly1305))
Provides: bundled(golang(golang.org/x/net/dns/dnsmessage))
Provides: bundled(golang(golang.org/x/net/http/httpguts))
Provides: bundled(golang(golang.org/x/net/http/httpproxy))
Provides: bundled(golang(golang.org/x/net/http2/hpack))
Provides: bundled(golang(golang.org/x/net/idna))
Provides: bundled(golang(golang.org/x/net/internal/nettest))
Provides: bundled(golang(golang.org/x/net/nettest))
Provides: bundled(golang(golang.org/x/text/secure))
Provides: bundled(golang(golang.org/x/text/secure/bidirule))
Provides: bundled(golang(golang.org/x/text/transform))
Provides: bundled(golang(golang.org/x/text/unicode))
Provides: bundled(golang(golang.org/x/text/unicode/bidi))
Provides: bundled(golang(golang.org/x/text/unicode/norm))
Provides: bundled(golang(github.com/google/pprof/driver))
Provides: bundled(golang(github.com/google/pprof/internal/binutils))
Provides: bundled(golang(github.com/google/pprof/internal/driver))
Provides: bundled(golang(github.com/google/pprof/internal/elfexec))
Provides: bundled(golang(github.com/google/pprof/internal/graph))
Provides: bundled(golang(github.com/google/pprof/internal/measurement))
Provides: bundled(golang(github.com/google/pprof/internal/plugin))
Provides: bundled(golang(github.com/google/pprof/internal/proftest))
Provides: bundled(golang(github.com/google/pprof/internal/report))
Provides: bundled(golang(github.com/google/pprof/internal/symbolizer))
Provides: bundled(golang(github.com/google/pprof/internal/symbolz))
Provides: bundled(golang(github.com/google/pprof/profile))
Provides: bundled(golang(github.com/google/pprof/third.party/d3))
Provides: bundled(golang(github.com/google/pprof/third.party/d3flamegraph))
Provides: bundled(golang(github.com/google/pprof/third.party/svgpan))
Provides: bundled(golang(github.com/ianlancetaylor/demangle))
Provides: bundled(golang(golang.org/x/arch/arm/armasm))
Provides: bundled(golang(golang.org/x/arch/arm64/arm64asm))
Provides: bundled(golang(golang.org/x/arch/ppc64/ppc64asm))
Provides: bundled(golang(golang.org/x/arch/x86/x86asm))
Provides: bundled(golang(golang.org/x/crypto/ssh/terminal))
Provides: bundled(golang(golang.org/x/sys/unix))
Provides: bundled(golang(golang.org/x/sys/windows))
Provides: bundled(golang(golang.org/x/sys/windows/registry))

Provides:       %{name}-bin = %{version}-%{release}
Obsoletes:      %{name}-bin
Obsoletes:      %{name}-shared
Obsoletes:      %{name}-docs
Obsoletes:      %{name}-data < 1.1.1-4
Obsoletes:      %{name}-vim < 1.4
Obsoletes:      emacs-%{name} < 1.4
Requires:       openEuler-rpm-config

ExclusiveArch:  %{golang_arches}

Patch6001:	0001-release-branch.go1.15-doc-go1.15-mention-1.15.3-cgo-.patch
Patch6002:	0002-release-branch.go1.15-cmd-go-fix-mod_get_fallback-te.patch
Patch6003:	0003-release-branch.go1.15-internal-execabs-only-run-test.patch
Patch6004:	0004-release-branch.go1.15-cmd-compile-don-t-short-circui.patch
Patch6005:	0005-release-branch.go1.15-cmd-go-fix-get_update_unknown_.patch
Patch6006:	0006-release-branch.go1.15-net-http-update-bundled-x-net-.patch
Patch6007:	0007-release-branch.go1.15-cmd-go-don-t-lookup-the-path-f.patch
Patch6008:	0008-release-branch.go1.15-cmd-link-internal-ld-pe-fix-se.patch
Patch6009:	0009-release-branch.go1.15-cmd-internal-goobj2-fix-buglet.patch
Patch6010:	0010-release-branch.go1.15-runtime-don-t-adjust-timer-pp-.patch
Patch6011:	0011-release-branch.go1.15-runtime-cgo-fix-Android-build-.patch
Patch6013:	0013-release-branch.go1.15-internal-poll-if-copy_file_ran.patch
Patch6014:	0014-release-branch.go1.15-internal-poll-netpollcheckerr-.patch
Patch6015:	0015-release-branch.go1.15-cmd-compile-do-not-assume-TST-.patch
Patch6016:	0016-release-branch.go1.15-syscall-do-not-overflow-key-me.patch
Patch6017:	0017-release-branch.go1.15-time-correct-unusual-extension.patch
Patch6018:	0018-release-branch.go1.15-cmd-compile-fix-escape-analysi.patch
Patch6019:	0019-release-branch.go1.15-net-http-ignore-connection-clo.patch
Patch6020:	0020-release-branch.go1.15-net-http-add-connections-back-.patch
Patch6021:	0021-release-branch.go1.15-security-encoding-xml-prevent-.patch
Patch6023:	0023-release-branch.go1.15-cmd-go-don-t-report-missing-st.patch
Patch6025:	0025-release-branch.go1.15-cmd-go-internal-modfetch-detec.patch
Patch6026:	0026-release-branch.go1.15-cmd-link-generate-trampoline-f.patch
Patch6027:	0027-release-branch.go1.15-net-http-update-bundled-x-net-.patch
Patch6028:	0028-release-branch.go1.15-net-http-fix-detection-of-Roun.patch
Patch6029:	0029-release-branch.go1.15-build-set-GOPATH-consistently-.patch
Patch6030:	0030-release-branch.go1.15-database-sql-fix-tx-stmt-deadl.patch
Patch6031:	0031-release-branch.go1.15-cmd-compile-disable-shortcircu.patch
Patch6032:	0032-release-branch.go1.15-runtime-non-strict-InlTreeInde.patch
Patch6033:	0033-release-branch.go1.15-cmd-cgo-avoid-exporting-all-sy.patch
Patch6034:	0034-release-branch.go1.15-cmd-link-avoid-exporting-all-s.patch
Patch6035:	0035-release-branch.go1.15-cmd-cgo-remove-unnecessary-spa.patch
Patch6037:	0037-release-branch.go1.15-time-use-offset-and-isDST-when.patch
Patch6038:	0038-release-branch.go1.15-std-update-golang.org-x-net-to.patch
Patch6039:	0039-release-branch.go1.15-runtime-time-disable-preemptio.patch
Patch6040:	0040-release-branch.go1.15-runtime-non-strict-InlTreeInde.patch
Patch6041:	0041-release-branch.go1.15-runtime-pprof-skip-tests-for-A.patch
Patch6043:	0043-release-branch.go1.15-math-big-fix-TestShiftOverlap-.patch
Patch6044:	0044-release-branch.go1.15-math-big-remove-the-s390x-asse.patch
Patch6045:	0045-net-http-fix-hijack-hang-at-abortPendingRead.patch
Patch6046:	0046-release-branch.go1.15-net-verify-results-from-Lookup.patch
Patch6047:	0047-release-branch.go1.15-archive-zip-only-preallocate-F.patch
Patch6048:	0048-release-branch.go1.15-net-http-httputil-always-remov.patch
Patch6049:	0049-release-branch.go1.15-math-big-check-for-excessive-e.patch
Patch6050:	0050-release-branch.go1.15-crypto-tls-test-key-type-when-.patch
Patch6051:	0051-net-reject-leading-zeros-in-IP-address-parsers.patch
Patch6052:	0052-release-branch.go1.16-misc-wasm-cmd-link-do-not-let-.patch
Patch6053:	0053-net-http-httputil-close-incoming-ReverseProxy-reques.patch
Patch6054:	0054-release-branch.go1.16-net-http-update-bundled-golang.patch
Patch6055:	0055-release-branch.go1.16-archive-zip-prevent-preallocat.patch
Patch6056:	0056-release-branch.go1.16-debug-macho-fail-on-invalid-dy.patch
Patch6057:	0057-release-branch.go1.16-math-big-prevent-overflow-in-R.patch
Patch6058:	0058-release-branch.go1.16-crypto-elliptic-make-IsOnCurve.patch
Patch6059:	0059-release-branch.go1.16-regexp-syntax-reject-very-deep.patch
Patch6060:	0060-cmd-go-internal-modfetch-do-not-short-circuit-canoni.patch
Patch6061:	0061-release-branch.go1.17-crypto-elliptic-tolerate-zero-.patch
Patch6062:	0062-release-branch.go1.17-encoding-pem-fix-stack-overflo.patch
Patch6063:	0063-release-branch.go1.16-syscall-fix-ForkLock-spurious-.patch
Patch6064:	0064-release-branch.go1.17-net-http-preserve-nil-values-i.patch
Patch6065:	0065-release-branch.go1.17-go-parser-limit-recursion-dept.patch
Patch6066:	0066-release-branch.go1.17-net-http-don-t-strip-whitespac.patch
Patch6067:	0067-release-branch.go1.17-encoding-xml-limit-depth-of-ne.patch
Patch6068:	0068-release-branch.go1.17-encoding-gob-add-a-depth-limit.patch
Patch6069:	0069-release-branch.go1.17-path-filepath-fix-stack-exhaus.patch
Patch6070:	0070-release-branch.go1.17-encoding-xml-use-iterative-Ski.patch
Patch6071:	0071-release-branch.go1.17-compress-gzip-fix-stack-exhaus.patch
Patch6072:	0072-release-branch.go1.17-crypto-tls-randomly-generate-t.patch
Patch6073:	0073-release-branch.go1.17-crypto-rand-properly-handle-la.patch
Patch6074:	0074-release-branch.go1.17-math-big-check-buffer-lengths-.patch
Patch6075:	0075-path-filepath-do-not-remove-prefix-.-when-following-.patch
Patch6076:	0076-release-branch.go1.17-syscall-check-correct-group-in.patch
Patch6077:	0077-release-branch.go1.16-runtime-consistently-access-po.patch
Patch6078:	0078-release-branch.go1.18-net-http-update-bundled-golang.patch
Patch6079:	0079-release-branch.go1.18-regexp-limit-size-of-parsed-re.patch
Patch6080:	0080-release-branch.go1.18-net-http-httputil-avoid-query-.patch
Patch6081:	0081-release-branch.go1.18-archive-tar-limit-size-of-head.patch
Patch6082:	0082-net-url-reject-query-values-with-semicolons.patch
Patch6083:	0083-syscall-os-exec-reject-environment-variables-contain.patch
Patch6084:	0084-release-branch.go1.18-net-http-update-bundled-golang.patch
Patch6085:	0085-all-update-vendored-golang.org-x-net.patch
Patch6086:	0086-release-branch.go1.19-crypto-tls-replace-all-usages-.patch
Patch6087:	0087-release-branch.go1.19-mime-multipart-limit-memory-in.patch
Patch6088:  0088-release-branch.go1.19-net-textproto-avoid-overpredic.patch
Patch6089:  0089-release-branch.go1.19-go-scanner-reject-large-line-a.patch
Patch6090:  0090-release-branch.go1.19-html-template-disallow-actions.patch
Patch6091:  0091-release-branch.go1.19-mime-multipart-avoid-excessive.patch
Patch6092:  0092-release-branch.go1.19-net-textproto-mime-multipart-i.patch
Patch6093:  0093-release-branch.go1.19-mime-multipart-limit-parsed-mi.patch

Patch9001:  0001-drop-hard-code-cert.patch
Patch9002:  0002-fix-patch-cmd-go-internal-modfetch-do-not-sho.patch

%description
%{summary}.

%package       help
Summary:       Golang compiler helps and manual docs
Requires:      %{name} = %{version}-%{release}
BuildArch:     noarch
Provides:      %{name}-docs = %{version}-%{release}
Obsoletes:     %{name}-docs
Provides:      %{name}-shared = %{version}-%{release}
Obsoletes:     %{name}-shared

%description   help
%{summary}.

%package       devel
Summary:       Golang compiler devel
BuildArch:     noarch
Requires:      %{name} = %{version}-%{release}
Provides:      %{name}-src  = %{version}-%{release}
Obsoletes:     %{name}-src
Provides:      %{name}-tests = %{version}-%{release}
Obsoletes:     %{name}-tests
Provides:      %{name}-misc = %{version}-%{release}
Obsoletes:     %{name}-misc
Obsoletes:     %{name}-race = %{version}-%{release}

%description   devel
%{summary}.

# Workaround old RPM bug of symlink-replaced-with-dir failure
%pretrans -p <lua>
for _,d in pairs({"api", "doc", "include", "lib", "src"}) do
  path = "%{goroot}/" .. d
  if posix.stat(path, "type") == "link" then
    os.remove(path)
    posix.mkdir(path)
  end
end

%prep
%autosetup -n go -p1

%build
uname -a
cat /proc/cpuinfo
cat /proc/meminfo

%if !%{golang_bootstrap}
export GOROOT_BOOTSTRAP=/
%else
export GOROOT_BOOTSTRAP=%{goroot}
%endif

export GOROOT_FINAL=%{goroot}
export GOHOSTOS=linux
export GOHOSTARCH=%{gohostarch}

pushd src
export CFLAGS="$RPM_OPT_FLAGS"
export LDFLAGS="$RPM_LD_FLAGS"
export CC="gcc"
export CC_FOR_TARGET="gcc"
export GOOS=linux
export GOARCH=%{gohostarch}
%if !%{external_linker}
export GO_LDFLAGS="-linkmode internal"
%endif
%if !%{cgo_enabled}
export CGO_ENABLED=0
%endif

%ifarch aarch64
export GO_LDFLAGS="-s -w"
%endif

./make.bash --no-clean -v
popd

%if %{shared}
GOROOT=$(pwd) PATH=$(pwd)/bin:$PATH go install -buildmode=shared -v -x std
%endif

%if %{race}
GOROOT=$(pwd) PATH=$(pwd)/bin:$PATH go install -race -v -x std
%endif

%install
rm -rf %{buildroot}
rm -rf pkg/obj/go-build/*

mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{goroot}

cp -apv api bin doc favicon.ico lib pkg robots.txt src misc test VERSION \
   %{buildroot}%{goroot}

# bz1099206
find %{buildroot}%{goroot}/src -exec touch -r %{buildroot}%{goroot}/VERSION "{}" \;
# and level out all the built archives
touch %{buildroot}%{goroot}/pkg
find %{buildroot}%{goroot}/pkg -exec touch -r %{buildroot}%{goroot}/pkg "{}" \;
# generate the spec file ownership of this source tree and packages
cwd=$(pwd)
src_list=$cwd/go-src.list
pkg_list=$cwd/go-pkg.list
shared_list=$cwd/go-shared.list
race_list=$cwd/go-race.list
misc_list=$cwd/go-misc.list
docs_list=$cwd/go-docs.list
tests_list=$cwd/go-tests.list
rm -f $src_list $pkg_list $docs_list $misc_list $tests_list $shared_list $race_list
touch $src_list $pkg_list $docs_list $misc_list $tests_list $shared_list $race_list
pushd %{buildroot}%{goroot}
    find src/ -type d -a \( ! -name testdata -a ! -ipath '*/testdata/*' \) -printf '%%%dir %{goroot}/%p\n' >> $src_list
    find src/ ! -type d -a \( ! -ipath '*/testdata/*' -a ! -name '*_test.go' \) -printf '%{goroot}/%p\n' >> $src_list

    find bin/ pkg/ -type d -a ! -path '*_dynlink/*' -a ! -path '*_race/*' -printf '%%%dir %{goroot}/%p\n' >> $pkg_list
    find bin/ pkg/ ! -type d -a ! -path '*_dynlink/*' -a ! -path '*_race/*' -printf '%{goroot}/%p\n' >> $pkg_list

    find doc/ -type d -printf '%%%dir %{goroot}/%p\n' >> $docs_list
    find doc/ ! -type d -printf '%{goroot}/%p\n' >> $docs_list

    find misc/ -type d -printf '%%%dir %{goroot}/%p\n' >> $misc_list
    find misc/ ! -type d -printf '%{goroot}/%p\n' >> $misc_list

%if %{shared}
    mkdir -p %{buildroot}/%{_libdir}/
    mkdir -p %{buildroot}/%{golibdir}/
    for file in $(find .  -iname "*.so" ); do
        chmod 755 $file
        mv  $file %{buildroot}/%{golibdir}
        pushd $(dirname $file)
        ln -fs %{golibdir}/$(basename $file) $(basename $file)
        popd
        echo "%%{goroot}/$file" >> $shared_list
        echo "%%{golibdir}/$(basename $file)" >> $shared_list
    done
    
	find pkg/*_dynlink/ -type d -printf '%%%dir %{goroot}/%p\n' >> $shared_list
	find pkg/*_dynlink/ ! -type d -printf '%{goroot}/%p\n' >> $shared_list
%endif

%if %{race}

    find pkg/*_race/ -type d -printf '%%%dir %{goroot}/%p\n' >> $race_list
    find pkg/*_race/ ! -type d -printf '%{goroot}/%p\n' >> $race_list

%endif

    find test/ -type d -printf '%%%dir %{goroot}/%p\n' >> $tests_list
    find test/ ! -type d -printf '%{goroot}/%p\n' >> $tests_list
    find src/ -type d -a \( -name testdata -o -ipath '*/testdata/*' \) -printf '%%%dir %{goroot}/%p\n' >> $tests_list
    find src/ ! -type d -a \( -ipath '*/testdata/*' -o -name '*_test.go' \) -printf '%{goroot}/%p\n' >> $tests_list
    # this is only the zoneinfo.zip
    find lib/ -type d -printf '%%%dir %{goroot}/%p\n' >> $tests_list
    find lib/ ! -type d -printf '%{goroot}/%p\n' >> $tests_list
popd

rm -rfv %{buildroot}%{goroot}/doc/Makefile

mkdir -p %{buildroot}%{goroot}/bin/linux_%{gohostarch}
ln -sf %{goroot}/bin/go %{buildroot}%{goroot}/bin/linux_%{gohostarch}/go
ln -sf %{goroot}/bin/gofmt %{buildroot}%{goroot}/bin/linux_%{gohostarch}/gofmt

mkdir -p %{buildroot}%{gopath}/src/github.com
mkdir -p %{buildroot}%{gopath}/src/bitbucket.org
mkdir -p %{buildroot}%{gopath}/src/code.google.com/p
mkdir -p %{buildroot}%{gopath}/src/golang.org/x

%check
export GOROOT=$(pwd -P)
export PATH="$GOROOT"/bin:"$PATH"
cd src

export CC="gcc"
export CFLAGS="$RPM_OPT_FLAGS"
export LDFLAGS="$RPM_LD_FLAGS"
%if !%{external_linker}
export GO_LDFLAGS="-linkmode internal"
%endif
%if !%{cgo_enabled} || !%{external_linker}
export CGO_ENABLED=0
%endif

export GO_TEST_TIMEOUT_SCALE=2

%if %{fail_on_tests}
echo tests ignored
%else
./run.bash --no-rebuild -v -v -v -k go_test:testing || :
%endif
cd ..

%post
%{_sbindir}/update-alternatives --install %{_bindir}/go \
    go %{goroot}/bin/go 90 \
    --slave %{_bindir}/gofmt gofmt %{goroot}/bin/gofmt

%preun
if [ $1 = 0 ]; then
    %{_sbindir}/update-alternatives --remove go %{goroot}/bin/go
fi

%if %{shared}
%files -f go-pkg.list -f go-shared.list
%else
%files -f go-pkg.list
%endif

%doc AUTHORS CONTRIBUTORS LICENSE PATENTS
%doc %{goroot}/VERSION
%dir %{goroot}/doc
%doc %{goroot}/doc/*
%dir %{goroot}
%exclude %{goroot}/src/
%exclude %{goroot}/doc/
%exclude %{goroot}/misc/
%exclude %{goroot}/test/
%exclude %{goroot}/lib/
%{goroot}/*
%dir %{gopath}
%dir %{gopath}/src
%dir %{gopath}/src/github.com/
%dir %{gopath}/src/bitbucket.org/
%dir %{gopath}/src/code.google.com/
%dir %{gopath}/src/code.google.com/p/
%dir %{gopath}/src/golang.org
%dir %{gopath}/src/golang.org/x

%files help -f go-docs.list

%files devel -f go-tests.list -f go-misc.list -f go-src.list

%changelog
* Thu Apr 13 2023 hanchao <hanchao47@huawei.com> - 1.15.7-25
- Type:CVE
- CVE:CVE-2023-24534,CVE-2023-24536,CVE-2023-24537,CVE-2023-24538
- SUG:NA
- DESC:fix CVE-2023-24534,CVE-2023-24536,CVE-2023-24537,CVE-2023-24538

* Thu Mar 23 2023 hanchao <hanchao47@huawei.com> - 1.15.7-24
- Type:CVE
- CVE:CVE-2022-41723,CVE-2022-41724,CVE-2022-41725
- SUG:NA
- DESC:fix CVE-2022-41723,CVE-2022-41724,CVE-2022-41725

* Fri Jan 20 2023 hanchao <hanchao47@huawei.com> - 1.15.7-23
- Type:CVE
- CVE:CVE-2022-41717
- SUG:NA
- DESC:fix CVE-2022-41717

* Thu Nov 17 2022 hanchao<hanchao47@huawei.com> - 1.15.7-22
- Type:CVE
- CVE:CVE-2022-41716
- SUG:NA
- DESC:fix CVE-2022-41716

* Mon Oct 10 2022 hanchao<hanchao47@huawei.com> - 1.15.7-21
- Type:CVE
- CVE:CVE-2022-41715,CVE-2022-2880,CVE-2022-2879
- SUG:NA
- DESC:fix CVE-2022-41715,CVE-2022-2880,CVE-2022-2879

* Wed Oct 05 2022 wangshuo <wangshuo@kylinos.cn> - 1.15.7-20
- Type:bugfix
- CVE:NA
- SUG:NA
- DESC:fix bad %goroot}/lib/ macro

* Thu Spe 15 2022 hanchao<hanchao47@huawei.com> - 1.15.7-19
- Type:CVE
- CVE:CVE-2022-27664
- SUG:NA
- DESC:fix CVE-2022-27664

* Thu Spe 8 2022 hanchao<hanchao47@huawei.com> - 1.15.7-18
- Type:bugfix
- CVE:NA
- SUG:NA
- DESC: runtime: consistently access pollDesc r/w Gs with atomics

* Tue Aug 30 2022 hanchao<hanchao47@huawei.com> - 1.15.7-17
- Type:bugfix
- CVE:NA
- SUG:NA
- DESC: golang: modify the golang.spec to remove unnecessary files
	from golang-help package

* Tue Aug 18 2022 hanchao<hanchao47@huawei.com> - 1.15.7-16
- fix CVE-2022-29804,CVE-2022-29526

* Mon Aug 8 2022 hanchao<hanchao47@huawei.com> - 1.15.7-15
- fix CVE-2022-32189

* Thu Jul 26 2022 hanchao<hanchao47@huawei.com> - 1.15.7-14
- fix CVE-2022-32148,CVE-2022-1962,CVE-2022-1705,CVE-2022-30633,
  CVE-2022-30635,CVE-2022-30632,CVE-2022-28131,
  CVE-2022-30631,CVE-2022-30629,CVE-2022-30634

* Thu May 12 2022 hanchao<hanchao47@huawei.com> - 1.15.7-13
- fix CVE-2021-44717

* Wed May 11 2022 hanchao<hanchao47@huawei.com> - 1.15.7-12
- fix CVE-2022-28327 CVE-2022-24675

* Thu Mar 24 2022 hanchao<hanchao47@huawei.com> - 1.15.7-11
- fix CVE-2022-23773

* Fri Mar 11 2022 hanchao<hanchao47@huawei.com> - 1.15.7-10
- fix CVE-2022-24921

* Fri Mar 4 2022 hanchao<hanchao47@huawei.com> - 1.15.7-9
- fix CVE-2022-23772  CVE-2022-23806

* Wed Mar 2 2022 hanchao<hanchao47@huawei.com> - 1.15.7-8
- fix CVE-2021-41771

* Tue Feb 8 2022 hanchao<hanchao47@huawei.com> - 1.15.7-7
- fix CVE-2021-39293

* Wed Jan 19 2022 hanchao<hanchao47@huawei.com> - 1.15.7-6
- fix CVE-2021-44716 

* Wed Oct 27 2021 chenjiankun <chenjiankun1@huawei.com> - 1.15.7-5
- fix CVE-2021-33195,CVE-2021-33196,CVE-2021-33197,CVE-2021-33198,CVE-2021-34558,CVE-2021-29923,CVE-2021-38297,CVE-2021-36221

* Fri Jun 18 2021 chenjiankun <chenjiankun1@huawei.com> - 1.15.7-4
- batch synchronization

* Fri Apr 23 2021 chenjiankun <chenjiankun1@huawei.com> - 1.15.7-3
- fix CVE-2021-27918

* Thu Apr 15 2021 lixiang <lixiang172@huawei.com> - 1.15.7-2
- speed up build progress

* Thu Jan 28 2021 xingweizheng <xingweizheng@huawei.com> - 1.15.7-1
- upgrade to 1.15.7

* Mon Jan 18 2021 jingrui<jingrui@huawei.com> - 1.13.15-2
- sync cve fix

* Tue Aug 18 2020 xiadanni <xiadanni1@huawei.com> - 1.13.15-1
- upgrade to 1.13.15

* Tue May 12 2020 lixiang <lixiang172@huawei.com> - 1.13.6
- rename tar name and make it same with upstream

* Tue Mar 17 2020 jingrui <jingrui@huawei.com> - 1.13.5
- drop hard code cert

* Mon Mar 23 2020 jingrui <jingrui@huawei.com> - 1.13.4
- fix CVE-2020-7919

* Thu Feb 20 2020 openEuler Buildteam <buildteam@openeuler.org> - 1.13-3.2
- requires remove mercurial

* Tue Dec 10 2019 jingrui<jingrui@huawei.com> - 1.13-3.1
- upgrade to golang 1.13.3

* Tue Sep 03 2019 leizhongkai<leizhongkai@huawei.com> - 1.11-1
- backport fix CVE-2019-9512 and CVE-2019-9514
