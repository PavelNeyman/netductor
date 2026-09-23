class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI workstation deploy)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.58"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.58/netductor-darwin-arm64"
      sha256 "61e99bc535fa6cae7ff8810389fd7a49e9d5ccbabb1814f25391afbbecda55d4"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.58/netductor-darwin-amd64"
      sha256 "07311f428a793d565804cc0e8392e829a5299bc4faa9a375a0e57db6c6ee71fb"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.58/netductor-linux-amd64"
      sha256 "6f34346dedf2a64cfd68a34497db3198caa62d0a96017a567f87ddc5221cdef8"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.58/netductor-linux-arm64"
      sha256 "c693a5bdac60b0f680c1897b8d204699ebfc8debf6b0867f3da5d8e2620867ee"
    end
  end

  head do
    url "https://github.com/PavelNeyman/netductor.git", branch: "main"
    depends_on "go" => :build
  end

  def install
    if build.head?
      system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=HEAD"), "./cmd/netductor"
    else
      bin.install Dir["netductor*"].first => "netductor"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
