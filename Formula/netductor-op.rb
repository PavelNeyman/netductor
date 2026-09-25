class NetductorOp < Formula
  desc "Netductor Mac operator (WebUI/TUI/deploy)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.15"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.15/netductor-op-darwin-arm64"
      sha256 "41e9069997d770b5b0d9881c68882a827b3f4b2b567e0df734474588942a043f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.15/netductor-op-darwin-amd64"
      sha256 "4bdcbf188280022047c9caea97513908d4972fc9780281891000dd9b61efea5d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.15/netductor-op-linux-amd64"
      sha256 "30fff57c771b4da8a3f33dc88d32e8ebd78605ba70ddd427b59b0747cd8a4779"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.15/netductor-op-linux-arm64"
      sha256 "f5b7dfc8f4ce511f5dacf5648d4cc4e541de6bf5762eec306a91cb0b5e56cd06"
    end
  end
  def install
    bin.install Dir["netductor-op*"].first => "netductor-op"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor-op version 2>&1")
  end
end
