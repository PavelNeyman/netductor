class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.84"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.84/netductor-darwin-arm64"
      sha256 "d24fbcddf4c95d8f1aeab6ad7ee8a6884c101eff2ced8d24253c1683ee263cb9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.84/netductor-darwin-amd64"
      sha256 "84bac5fc02568d1fd50eb25016b03ed0b95dbf077a92f3332fecca449330b35a"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.84/netductor-linux-amd64"
      sha256 "72f49eaa3d64666f79b866c730af8d69b4b1d2880114d19d5ba83850b3614a89"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.84/netductor-linux-arm64"
      sha256 "4a6420cc93446aa674ccf3cfefac37e083a965d524506a0ba3e19d17026dec8d"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
