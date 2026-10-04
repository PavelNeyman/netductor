class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.217"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.217/netductor-op-darwin-arm64"
      sha256 "fc5e02986a83971ef636b2da84e490e758cf94be79eacbd2971c8684f52c0bb7"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.217/netductor-op-darwin-amd64"
      sha256 "17293956286bd5d6b01f0010b145f7f8f3d610ab6c8949bd490c1213b0e9cb39"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.217/netductor-op-linux-amd64"
      sha256 "07153c0bcca7dec1c87b3e90cb7fc91df936b27db0b77aba0072f1693beb1906"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
