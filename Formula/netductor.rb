class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI workstation deploy)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.57"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.57/netductor-darwin-arm64"
      sha256 "fdcb4d880d7e2970e5342aeea3cb588284374ee27438dc142f7277911db11336"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.57/netductor-darwin-amd64"
      sha256 "5551695faaa1d057e0eb57f2f7a1aa610fcfc7e48b4d0bfed076192fc489f8ad"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.57/netductor-linux-amd64"
      sha256 "4bfc0fac5cd78137a746065bc91b61eb213f4d6cb5259652acf4b6c26ea3201f"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.57/netductor-linux-arm64"
      sha256 "a3681d21f7146987f59d293e6534eeaa452532c21db4e0980b07b6ca547fe133"
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
