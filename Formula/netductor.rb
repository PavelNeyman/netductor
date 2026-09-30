class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.141"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.141/netductor-op-darwin-arm64"
      sha256 "5d5fccbe6aa54fc62061d0bea52b405ccd58f6e3f7340fe05a1205907b94de13"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.141/netductor-op-linux-amd64"
      sha256 "729e1842a60ee27d09a08abf28b42164937f118e0c5e99a91f0996093c3629ee"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
