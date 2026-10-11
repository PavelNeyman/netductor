class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.335"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.335/netductor-op-darwin-arm64"
      sha256 "bc79a35458ecd4d70429421d6319a35fb0ef9123eacc68a725ea0f90ddd57095"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.335/netductor-op-darwin-amd64"
      sha256 "f9044740a7ec9562799c6ceeecaede50139de9773b1b2b0c1a5242abc0626709"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.335/netductor-op-linux-amd64"
      sha256 "503179c40bcdee5d1e90e0774cc1d846111927ed32f016a75c2aeea0529acf67"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
