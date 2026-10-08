class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.300"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.300/netductor-op-darwin-arm64"
      sha256 "8b3424944b1cbbf48b00deff27601687f569bc6e13de807d723d6f66589059c3"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.300/netductor-op-darwin-amd64"
      sha256 "150614223c17eccb66ebd4f79bc373e06e9d1791cd123d6bf8b20cf1bcdeaa87"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.300/netductor-op-linux-amd64"
      sha256 "b338f7a44a432a5c187154a465a1bc22b015f6dc35a6344a027b62aa5b6d2b3f"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
