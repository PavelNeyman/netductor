class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.395"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.395/netductor-op-darwin-arm64"
      sha256 "692cfa2270410ca70d2089d1227df1490b711128f807410b4d8c35fba041cedd"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.395/netductor-op-darwin-amd64"
      sha256 "e00c9d5bd5e77db885b38d89008297455bfeba2362dfea571b49294bb17dd51d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.395/netductor-op-linux-amd64"
      sha256 "5cafd2b31ea54b498e25c0d3706bc182a3e0fd8683f24a6cab03f460470447b3"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
