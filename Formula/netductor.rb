class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.89"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.89/netductor-darwin-arm64"
      sha256 "0e0f18b16116f469ae0626bccd832cea7f7f38fd5fca67d972825f6b70d6249a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.89/netductor-darwin-amd64"
      sha256 "d7125f58028b31412cf60e2ae2c5891f5276f4be0cfc9f9228bb8fb02a9a5ab1"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.89/netductor-linux-amd64"
      sha256 "a52b08e73cfca754f2561621b0b8a93f376c7a019d808eda6719a045eedead4a"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.89/netductor-linux-arm64"
      sha256 "7ab62209758fa12b3bfa7195b9b5ead32aba6c4a400e18a322b1715d40ae44e5"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
