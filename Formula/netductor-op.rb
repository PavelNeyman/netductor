class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.326"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.326/netductor-op-darwin-arm64"
      sha256 "783ee4d4b74eadaeab3223d793af60fb69e4e4c36f8359c8e42be240e1f105ac"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.326/netductor-op-darwin-amd64"
      sha256 "2588e32dc777ed71bc338cba806d2b5fe3e7830c79c3faaf553ac43c9dea0dfc"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.326/netductor-op-linux-amd64"
      sha256 "d33f893cfa5ba4e815cfd017a974028d71ca788a2e738b60473c4fec090edf19"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
