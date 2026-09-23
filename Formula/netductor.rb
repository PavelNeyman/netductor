class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI workstation deploy)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.60"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.60/netductor-darwin-arm64"
      sha256 "63c4207373e03a37ffe87faae0bf68eb12a315647cf159fac25efb85da2515c9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.60/netductor-darwin-amd64"
      sha256 "03be59187b54575efa6bfb9329ef452e2c47b9089f06c4ad856dcb42233029b0"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.60/netductor-linux-amd64"
      sha256 "b54d91f1c37e04ef9fe3f5cf851706d7114cfb7bb30a470c3933f556a1c67763"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.60/netductor-linux-arm64"
      sha256 "99f394a40d4d60a82159f9429e554ac43538509dfeb475a17d6938e36bcf57e9"
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
