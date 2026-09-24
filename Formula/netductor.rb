class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.95"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.95/netductor-darwin-arm64"
      sha256 "22208aa87fdca0b04203d943b42a5826e7f99aa456050d16a7d26c81f223b2e9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.95/netductor-darwin-amd64"
      sha256 "6cb63e1d06bf058cf60b2e942bba16fc28ed5fdc27f13e28c12bfceb05a2ab17"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.95/netductor-linux-amd64"
      sha256 "60d336ad2662cb6709852213fca40c254bc5db1d6f23aa77171cf5c93679aba3"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.95/netductor-linux-arm64"
      sha256 "5a97160281b37acbd3505eb91e05b4fe31418b5d24170af3bdc1a3a750291aa9"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
