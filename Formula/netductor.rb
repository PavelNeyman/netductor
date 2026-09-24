class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.86"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.86/netductor-darwin-arm64"
      sha256 "de15053d2a31b91a339da7096a513d55201b5be65f97c483e0df2469d86dafd6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.86/netductor-darwin-amd64"
      sha256 "b039198521b8a9039bee76888ecd623d69e5322bf9df83afd27d521cf17e4079"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.86/netductor-linux-amd64"
      sha256 "7b56f9de3c77581c3dd43c78209c03c9dab02c2be1856c8c314f60eba1d4bca6"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.86/netductor-linux-arm64"
      sha256 "07cec6ecb721f2e24a772d3729c43d6c1e56c5e17e94c0796c6dd91a168fe7f2"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
