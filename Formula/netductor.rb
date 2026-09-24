class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.91"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.91/netductor-darwin-arm64"
      sha256 "6795f441f5f494b7ea5244ff19481b17b25389623d8d3d1f21c06dd9323c48c6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.91/netductor-darwin-amd64"
      sha256 "0ec6070e30d921d0d7fe974127114673f457d80795bf422f8e93c868b733b972"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.91/netductor-linux-amd64"
      sha256 "becceacdf28368fb72b4c5e69b6fdfe85602f00bf964e51c3ec652eee16f8472"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.91/netductor-linux-arm64"
      sha256 "8345dda077883c8772bf558bb19f5e90fb91758d889eda26db89aa809e9bf7fc"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
