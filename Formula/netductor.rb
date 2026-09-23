class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.69"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.69/netductor-darwin-arm64"
      sha256 "0be01aa03879117d908ba2f19b14322543421a43be3af4ba862f064940bd024e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.69/netductor-darwin-amd64"
      sha256 "d9d16cc667ccfcdfc53e1353179d4c4b568816078c11fc0df332f011cbfbf00c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.69/netductor-linux-amd64"
      sha256 "628e9f1f5e3d581bbdec1b15485ffe07f51539dcc9c885e8b79171670257ab2c"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.69/netductor-linux-arm64"
      sha256 "fc91746122fbb87e5dc6979768c65c9bf22bb161ea367eaeda496dcec97ad2e5"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
