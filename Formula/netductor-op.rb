class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.261"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.261/netductor-op-darwin-arm64"
      sha256 "d0db24a09b65b5dc2aa7de1ff91f34aeb2f4144680619fd534e444b827dabc78"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.261/netductor-op-darwin-amd64"
      sha256 "05532a896576bcc8c312444cfa4bfc392c57e25e66ed65d864e85afc8bf9ae82"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.261/netductor-op-linux-amd64"
      sha256 "6cf417bccea9cbbaeb4ac5d20a1747ac8ac5bc544ad0993747892ad29412b6c6"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
