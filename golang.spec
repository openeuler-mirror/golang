%global debug_package %{nil}
%global _binaries_in_noarch_packages_terminate_build 0
%global golibdir %{_libdir}/golang
%global goroot /usr/lib/%{name}
%global go_api 1.17
%global go_version 1.17
%global __spec_install_post /usr/lib/rpm/check-rpaths /usr/lib/rpm/check-buildroot /usr/lib/rpm/brp-compress
%global __requires_exclude_from ^(%{_datadir}|/usr/lib)/%{name}/(doc|src)/.*$
%global __strip /bin/true
%define _use_internal_dependency_generator 0
%define __find_requires %{nil}

%bcond_with bootstrap
%ifarch x86_64 aarch64 riscv64
%bcond_without ignore_tests
%else
%bcond_with ignore_tests
%endif

%ifarch x86_64 aarch64 riscv64
%global external_linker 1
%else
%global external_linker 0
%endif

%ifarch x86_64 aarch64 riscv64
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

%ifarch x86_64 aarch64
%global shared 1
%else
%global shared 0
%endif

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
%ifarch riscv64
%global gohostarch riscv64
%endif


Name:           golang
Version:        1.17.3
Release:        1
Summary:        The Go Programming Language
License:        BSD and Public Domain
URL:            https://golang.org/
Source0:        https://dl.google.com/go/go1.17.3.src.tar.gz

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

cp -apv api bin doc lib pkg src misc test VERSION \
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

%files -f go-pkg.list
%doc AUTHORS CONTRIBUTORS LICENSE PATENTS
%doc %{goroot}/VERSION
%dir %{goroot}/doc
%doc %{goroot}/doc/*
%dir %{goroot}
%exclude %{goroot}/src/
%exclude %{goroot}/doc/
%exclude %{goroot}/misc/
%exclude %{goroot}/test/
%{goroot}/*
%dir %{gopath}
%dir %{gopath}/src
%dir %{gopath}/src/github.com/
%dir %{gopath}/src/bitbucket.org/
%dir %{gopath}/src/code.google.com/
%dir %{gopath}/src/code.google.com/p/
%dir %{gopath}/src/golang.org
%dir %{gopath}/src/golang.org/x

%if %{shared}
%files help -f go-docs.list -f go-shared.list
%endif

%files devel -f go-tests.list -f go-misc.list -f go-src.list

%changelog
* Mon Nov 29 2021 chenjiankun <chenjiankun1@huawei.com> - 1.17.3-1
- upgrade to 1.17.3

* Thu Apr 15 2021 lixiang <lixiang172@huawei.com> - 1.15.7-2
- speed up build progress

* Thu Jan 28 2021 xingweizheng <xingweizheng@huawei.com> - 1.15.7-1
- upgrade to 1.15.7

* Mon Dec 7 2020 yangyanchao <yangyanchao6@huawei.com> - 1.15.5-3
- Enable Cgo for RISC-V

* Sat Nov 28 2020 whoisxxx <zhangxuzhou4@huawei.com> - 1.15.5-2
- Adate for RISC-V

* Wed Nov 18 2020 liuzekun <liuzekun@huawei.com> - 1.15.5-1
- upgrade to 1.15.5

* Tue Aug 18 2020 xiadanni <xiadanni1@huawei.com> - 1.13.15-1
- upgrade to 1.13.15

* Fri Jul 31 2020 xiadanni <xiadanni1@huawei.com> - 1.13.14-2
- add yaml file

* Thu Jul 30 2020 xiadanni <xiadanni1@huawei.com> - 1.13.14-1
- upgrade to 1.13.14

* Thu Jul 23 2020 xiadanni <xiadanni1@huawei.com> - 1.13-4.1
- bump to 1.13.4

* Tue May 12 2020 lixiang <lixiang172@huawei.com> - 1.13-3.6
- rename tar name and make it same with upstream

* Tue Mar 24 2020 jingrui <jingrui@huawei.com> - 1.13-3.5
- drop hard code cert

* Mon Mar 23 2020 jingrui <jingrui@huawei.com> - 1.13-3.4
- fix CVE-2020-7919

* Thu Feb 20 2020 openEuler Buildteam <buildteam@openeuler.org> - 1.13-3.2
- requires remove mercurial

* Tue Dec 10 2019 jingrui<jingrui@huawei.com> - 1.13-3.1
- upgrade to golang 1.13.3

* Tue Sep 03 2019 leizhongkai<leizhongkai@huawei.com> - 1.11-1
- backport fix CVE-2019-9512 and CVE-2019-9514
