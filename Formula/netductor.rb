class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.145"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.145/netductor-op-darwin-arm64"
      sha256 "d117b7caa68b8e1635ea6785b00235df1c4de018c72188360266300957170f23"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.145/netductor-op-linux-amd64"
      sha256 "bcc6b3e5903645b11ef5c3cf8d6c1b1ca0a890f6ee8d47a8be2a9eba308d406c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
