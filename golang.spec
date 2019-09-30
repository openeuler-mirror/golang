%bcond_with bootstrap
# temporalily ignore test failures
%ifarch x86_64 aarch64
%bcond_without ignore_tests
%else
%bcond_with ignore_tests
%endif

# build ids are not currently generated:
# https://code.google.com/p/go/issues/detail?id=5238
#
# also, debuginfo extraction currently fails with
# "Failed to write file: invalid section alignment"
%global debug_package %{nil}

# we are shipping the full contents of src in the data subpackage, which
# contains binary-like things (ELF data for tests, etc)
%global _binaries_in_noarch_packages_terminate_build 0

# Do not check any files in doc or src for requires
%global __requires_exclude_from ^(%{_datadir}|/usr/lib)/%{name}/(doc|src)/.*$

# Don't alter timestamps of especially the .a files (or else go will rebuild later)
# Actually, don't strip at all since we are not even building debug packages and this corrupts the dwarf testdata
%global __strip /bin/true

# rpmbuild magic to keep from having meta dependency on libc.so.6
%define _use_internal_dependency_generator 0
%define __find_requires %{nil}
%global __spec_install_post /usr/lib/rpm/check-rpaths   /usr/lib/rpm/check-buildroot  \
  /usr/lib/rpm/brp-compress

%global golibdir %{_libdir}/golang

# Golang build options.

# Build golang using external/internal(close to cgo disabled) linking.
%ifarch x86_64 aarch64
%global external_linker 1
%else
%global external_linker 0
%endif

# Build golang with cgo enabled/disabled(later equals more or less to internal linking).
%ifarch x86_64 aarch64
%global cgo_enabled 1
%else
%global cgo_enabled 0
%endif

# Use golang/gcc-go as bootstrap compiler
%if %{with bootstrap}
%global golang_bootstrap 0
%else
%global golang_bootstrap 1
%endif

# Controls what ever we fail on failed tests
%if %{with ignore_tests}
%global fail_on_tests 0
%else
%global fail_on_tests 1
%endif

# Build golang shared objects for stdlib
%ifarch x86_64 aarch64
%global shared 1
%else
%global shared 0
%endif

# Pre build std lib with -race enabled
%ifarch x86_64
%global race 1
%else
%global race 0
%endif

%global goroot          /usr/lib/%{name}

%ifarch x86_64
%global gohostarch  amd64
%endif
%ifarch aarch64
%global gohostarch  arm64
%endif

%global go_api 1.11
%global go_version 1.11

Name:           golang
Version:        1.11
Release:        4
Summary:        The Go Programming Language
# source tree includes several copies of Mark.Twain-Tom.Sawyer.txt under Public Domain
License:        BSD and Public Domain
URL:            http://golang.org/
Source0:        https://storage.googleapis.com/golang/go%{go_version}.src.tar.gz
# make possible to override default traceback level at build time by setting build tag rpm_crashtraceback

# The compiler is written in Go. Needs go(1.4+) compiler for build.
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
# Pre-go1.5, all arches had to be bootstrapped individually, before usable, and
# env variables to compile for the target os-arch.
# Now the host compiler needs only the GOOS and GOARCH environment variables
# set to compile for the target os-arch.
Obsoletes:      %{name}-pkg-bin-linux-386 < 1.4.99
Obsoletes:      %{name}-pkg-bin-linux-amd64 < 1.4.99
Obsoletes:      %{name}-pkg-bin-linux-arm < 1.4.99
Obsoletes:      %{name}-pkg-linux-386 < 1.4.99
Obsoletes:      %{name}-pkg-linux-amd64 < 1.4.99
Obsoletes:      %{name}-pkg-linux-arm < 1.4.99
Obsoletes:      %{name}-pkg-darwin-386 < 1.4.99
Obsoletes:      %{name}-pkg-darwin-amd64 < 1.4.99
Obsoletes:      %{name}-pkg-windows-386 < 1.4.99
Obsoletes:      %{name}-pkg-windows-amd64 < 1.4.99
Obsoletes:      %{name}-pkg-plan9-386 < 1.4.99
Obsoletes:      %{name}-pkg-plan9-amd64 < 1.4.99
Obsoletes:      %{name}-pkg-freebsd-386 < 1.4.99
Obsoletes:      %{name}-pkg-freebsd-amd64 < 1.4.99
Obsoletes:      %{name}-pkg-freebsd-arm < 1.4.99
Obsoletes:      %{name}-pkg-netbsd-386 < 1.4.99
Obsoletes:      %{name}-pkg-netbsd-amd64 < 1.4.99
Obsoletes:      %{name}-pkg-netbsd-arm < 1.4.99
Obsoletes:      %{name}-pkg-openbsd-386 < 1.4.99
Obsoletes:      %{name}-pkg-openbsd-amd64 < 1.4.99

Obsoletes:      golang-vet < 0-12.1
Obsoletes:      golang-cover < 0-12.1

Requires(post): %{_sbindir}/update-alternatives
Requires(postun): %{_sbindir}/update-alternatives

# We strip the meta dependency, but go does require glibc.
# This is an odd issue, still looking for a better fix.
Requires:       glibc
Requires:       gcc
Requires:       git, subversion, mercurial



# Bundled/Vendored provides generated by
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
Requires:       go-srpm-macros

Patch1:       0001-Don-t-use-the-bundled-tzdata-at-runtime-except-for-t.patch
Patch2:       0002-syscall-expose-IfInfomsg.X__ifi_pad-on-s390x.patch
Patch3:       0003-release-branch.go1.11-security-crypto-elliptic-reduc.patch
Patch4:       0004-fix-CVE-2019-9512-9514.patch

# Having documentation separate was broken
Obsoletes:      %{name}-docs

# RPM can't handle symlink -> dir with subpackages, so merge back
Obsoletes:      %{name}-data < 1.1.1-4

# go1.4 deprecates a few packages
Obsoletes:      %{name}-vim < 1.4
Obsoletes:      emacs-%{name} < 1.4

ExclusiveArch:  %{golang_arches}

Source100:      golang-gdbinit

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
Requires:      %{name} = %{version}-%{release}
BuildArch:     noarch
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
# print out system information
uname -a
cat /proc/cpuinfo
cat /proc/meminfo

# bootstrap compiler GOROOT
%if !%{golang_bootstrap}
export GOROOT_BOOTSTRAP=/
%else
export GOROOT_BOOTSTRAP=%{goroot}
%endif

# set up final install location
export GOROOT_FINAL=%{goroot}

export GOHOSTOS=linux
export GOHOSTARCH=%{gohostarch}

pushd src
# use our gcc options for this build, but store gcc as default for compiler
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

# build shared std lib
%if %{shared}
GOROOT=$(pwd) PATH=$(pwd)/bin:$PATH go install -buildmode=shared -v -x std
%endif

%if %{race}
GOROOT=$(pwd) PATH=$(pwd)/bin:$PATH go install -race -v -x std
%endif

%install
rm -rf $RPM_BUILD_ROOT
# remove GC build cache
rm -rf pkg/obj/go-build/*

# create the top level directories
mkdir -p $RPM_BUILD_ROOT%{_bindir}
mkdir -p $RPM_BUILD_ROOT%{goroot}

# install everything into libdir (until symlink problems are fixed)
# https://code.google.com/p/go/issues/detail?id=5830
cp -apv api bin doc favicon.ico lib pkg robots.txt src misc test VERSION \
   $RPM_BUILD_ROOT%{goroot}

# bz1099206
find $RPM_BUILD_ROOT%{goroot}/src -exec touch -r $RPM_BUILD_ROOT%{goroot}/VERSION "{}" \;
# and level out all the built archives
touch $RPM_BUILD_ROOT%{goroot}/pkg
find $RPM_BUILD_ROOT%{goroot}/pkg -exec touch -r $RPM_BUILD_ROOT%{goroot}/pkg "{}" \;
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
pushd $RPM_BUILD_ROOT%{goroot}
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

# remove the doc Makefile
rm -rfv $RPM_BUILD_ROOT%{goroot}/doc/Makefile

# put binaries to bindir, linked to the arch we're building,
# leave the arch independent pieces in {goroot}
mkdir -p $RPM_BUILD_ROOT%{goroot}/bin/linux_%{gohostarch}
ln -sf %{goroot}/bin/go $RPM_BUILD_ROOT%{goroot}/bin/linux_%{gohostarch}/go
ln -sf %{goroot}/bin/gofmt $RPM_BUILD_ROOT%{goroot}/bin/linux_%{gohostarch}/gofmt

# ensure these exist and are owned
mkdir -p $RPM_BUILD_ROOT%{gopath}/src/github.com
mkdir -p $RPM_BUILD_ROOT%{gopath}/src/bitbucket.org
mkdir -p $RPM_BUILD_ROOT%{gopath}/src/code.google.com/p
mkdir -p $RPM_BUILD_ROOT%{gopath}/src/golang.org/x

# gdbinit
mkdir -p $RPM_BUILD_ROOT%{_sysconfdir}/gdbinit.d
cp -av %{SOURCE100} $RPM_BUILD_ROOT%{_sysconfdir}/gdbinit.d/golang.gdb

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

# make sure to not timeout
export GO_TEST_TIMEOUT_SCALE=2

%if %{fail_on_tests}
# ./run.bash --no-rebuild -v -v -v -k
echo tests ignored
%else
./run.bash --no-rebuild -v -v -v -k || :
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
# VERSION has to be present in the GOROOT, for `go install std` to work
%doc %{goroot}/VERSION
%dir %{goroot}/doc
%doc %{goroot}/doc/*

# go files
%dir %{goroot}
%exclude %{goroot}/src/
%exclude %{goroot}/doc/
%exclude %{goroot}/misc/
%exclude %{goroot}/test/
%{goroot}/*

# ensure directory ownership, so they are cleaned up if empty
%dir %{gopath}
%dir %{gopath}/src
%dir %{gopath}/src/github.com/
%dir %{gopath}/src/bitbucket.org/
%dir %{gopath}/src/code.google.com/
%dir %{gopath}/src/code.google.com/p/
%dir %{gopath}/src/golang.org
%dir %{gopath}/src/golang.org/x

# gdbinit (for gdb debugging)
%{_sysconfdir}/gdbinit.d

%files help -f go-docs.list -f go-shared.list

%files devel -f go-tests.list -f go-misc.list -f go-src.list

%changelog
* Tue Sep 03 2019 leizhongkai<leizhongkai@huawei.com> - 1.11-1
- backport fix CVE-2019-9512 and CVE-2019-9514
