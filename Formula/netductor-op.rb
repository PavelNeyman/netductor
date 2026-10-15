class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.396"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.396/netductor-op-darwin-arm64"
      sha256 "36d2ba772c4d099989ff4bb1b7454e9c3c352056e3a583993220d0af8e894744"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.396/netductor-op-darwin-amd64"
      sha256 "1760cc13411b510cd5fd0052c03964837df5eaa66e42baf5229c3487bd8c1218"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.396/netductor-op-linux-amd64"
      sha256 "1e04543a17368628ca3481bc9be7fc129805259f066786e8f3c3acc40b8d96c1"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
