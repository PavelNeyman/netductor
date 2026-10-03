class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.207"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.207/netductor-op-darwin-arm64"
      sha256 "9a4b0f07a98ffab5d394a345197398a36444fb6c2da17bb69069c845f0fda586"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.207/netductor-op-darwin-amd64"
      sha256 "ebe24e036cd2b63fdc99859c9b46b960c349973e842ee547a0b43f49669c6303"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.207/netductor-op-linux-amd64"
      sha256 "ff111c1af51b361ebfe0174290fc584ab94c20ced323da2ea3e1f2e7ed4501eb"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
