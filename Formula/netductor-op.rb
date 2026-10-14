class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.389"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.389/netductor-op-darwin-arm64"
      sha256 "763fe738cb7b6e0995e9c6f4452cc6025e7c6b7d4e7e698bdfc9a9bf9333bc32"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.389/netductor-op-darwin-amd64"
      sha256 "359f496a7cb7141018a70c15063d862f1a67bf5b71fc9382932291ad5aa3f91d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.389/netductor-op-linux-amd64"
      sha256 "d1509dadf3e15b89e007d8730ce3e1dc0a7522554326cfa08f896ceae560dc1c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
