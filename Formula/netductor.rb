class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI workstation deploy)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.59"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.59/netductor-darwin-arm64"
      sha256 "4e702b1f398164b90fdb106e5f94871c65e570b0392d5578efcc6b37e1169fc4"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.59/netductor-darwin-amd64"
      sha256 "ecbaf355707205c480c360d2fb7d8fe0c4f10f8d5b9faa728ba6350079b06fc4"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.59/netductor-linux-amd64"
      sha256 "8aac8ef8293acfc7b8561c35fb30b539b06c87e990a32e9ce88f2e655da3bfc6"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.59/netductor-linux-arm64"
      sha256 "a2de0715ea65107ee02035604f5bd9848ae7f8667e863134180159675a512b40"
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
