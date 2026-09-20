class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI workstation deploy)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.3"
  license "MIT"

  # Prefer prebuilt release assets; fall back to HEAD source build.
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.3/netductor-darwin-arm64"
      sha256 :no_check
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.3/netductor-darwin-amd64"
      sha256 :no_check
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.3/netductor-linux-amd64"
      sha256 :no_check
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.3/netductor-linux-arm64"
      sha256 :no_check
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
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
