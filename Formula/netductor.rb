class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.3"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.3/netductor-op-darwin-arm64"
      sha256 "25b2d43b1a110ff8ba8f82e26d8586cf4d0b490115d4d791471e296817c65825"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.3/netductor-op-darwin-amd64"
      sha256 "012cefeb00e8f58e79c9ba72cbbae229c4eceb4edd8bbd8e34f0c9a0f373c1d5"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.3/netductor-op-linux-amd64"
      sha256 "46fea436ca7376f8d76a41c833cfef98b7bd7eada9c7cd6a6c8a3378049e681c"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.3/netductor-op-linux-arm64"
      sha256 "18c9436b3273bfe786d85c5d242a8439befd548028d6887837e658b497a294a5"
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
