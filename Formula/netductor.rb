class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.4"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.4/netductor-op-darwin-arm64"
      sha256 "64eeb779cba6ade81dab122cc9db2772b99fd0a170bbc2b939f58093ac9e0d3c"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.4/netductor-op-darwin-amd64"
      sha256 "5ae3ff8a94284846a8ca7dfa240177aa482256a97fb1e1e31d0c490c1df9273a"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.4/netductor-op-linux-amd64"
      sha256 "a118d2513ce478d5120c2f476f7a453693bcddbbd6f20a8efefa46187c5469f0"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.4/netductor-op-linux-arm64"
      sha256 "a601e0a76c84b019f9bcb479f17c0f67eb22f6ffbe5a671719985d4125f2e6de"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
