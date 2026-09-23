class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.76"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.76/netductor-darwin-arm64"
      sha256 "5f96139d07f651bed28b4ef5896fc79810e6e0e98942a572e68af6e25da0a078"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.76/netductor-darwin-amd64"
      sha256 "8eaaad47ae2469a580b9e9cd7bc82f6663af57ada39e9cf8985fd9fb82f46702"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.76/netductor-linux-amd64"
      sha256 "933c228252132e657905f535c2c6ffbb10b3208f5c984efa57d7ec72adedc330"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.76/netductor-linux-arm64"
      sha256 "3e9357ac010aba68496a804624a7d1f480c7aa610f1195dc6a41e86ee869f163"
    end
  end

  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
