class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.444"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.444/netductor-op-darwin-arm64"
      sha256 "e16db368198778857247c98a6c650553bb93fec356b71cd8e5d29d18ef5ed077"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.444/netductor-op-darwin-amd64"
      sha256 "22809a9ac8d361d799caa6c07a2f548dd03a542b8cfb618aa8ad081ba3859af3"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.444/netductor-op-linux-amd64"
      sha256 "e5282536745db97e1438808b9f282aaacee0cd6bd610af455679bb25c9d8d071"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
