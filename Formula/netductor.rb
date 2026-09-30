class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.125"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.125/netductor-op-darwin-arm64"
      sha256 "b46cdcae6f3b3235163dd43e03788377f81c5dcf5767272c1e399392a84077d7"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.125/netductor-op-darwin-amd64"
      sha256 "7f41ebf79294f9a8a8af7120063a26f352c0453649c146f4940e526e8fd20e8b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.125/netductor-op-linux-amd64"
      sha256 "c6314abdc51427357c41f757340304a8f0314ff80f03b95e5e6236b16475512d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
