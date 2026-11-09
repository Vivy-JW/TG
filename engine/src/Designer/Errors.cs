using System;

namespace TzsCli.Designer {

    /// <summary>
    /// A caller error that the operation layer can describe and the transport can map to a
    /// wire error code. Exists so the library never has to end the process to report one.
    ///
    /// This is not style. The original Edit.exe called Environment.Exit(2) inline when a path
    /// did not resolve, which is fine for a one-shot exe but would kill the long-lived server
    /// on the first bad argument -- and a daemon that dies on bad input is worse than one that
    /// answers "no such path". SPEC §11.24 (a) maps these to error.kind in the response.
    ///
    /// Codes here are the wire codes from §11.24 (a); keeping them in the exception means the
    /// JSON-RPC layer does not need a translation table that could drift from the throw sites.
    /// </summary>
    public class TzsError : Exception {
        public readonly string Code;

        public TzsError(string code, string message) : base(message) { Code = code; }

        /// <summary>The caller named something that does not resolve. They can self-correct.</summary>
        public static TzsError NotFound(string what, string value) {
            return new TzsError("not_found", what + " 不存在: " + value);
        }

        /// <summary>The caller's argument is malformed -- wrong type, bad enum, unknown name.</summary>
        public static TzsError Validation(string message) {
            return new TzsError("validation", message);
        }

        /// <summary>The handle is gone: never existed, or closed. Wire code E_NO_HANDLE, kind
        /// not_found (Rpc.Map has the case) -- the caller's fix is "open it again", so this must
        /// NOT be the "don't retry" kind it used to be reported as.
        ///
        /// <paramref name="what"/> stays a parameter so the two situations keep reading differently
        /// ("句柄" vs "句柄（已关闭）") while sharing one code.</summary>
        public static TzsError NoHandle(string what, string handle) {
            return new TzsError("E_NO_HANDLE", what + " 不存在: " + handle);
        }

        /// <summary>A name-path that resolves to nothing. Wire code E_PATH_NOT_FOUND, kind
        /// not_found: the caller fixes the path and retries, which is why it is not E_DESIGNER.
        ///
        /// Six call sites used to say `NotFound("路径", path)`, i.e. E_NOT_FOUND -- correct in
        /// kind, but it left the caller to guess whether the thing that was not found was the path,
        /// the table or the handle. §11.24 (a) lists E_PATH_NOT_FOUND for exactly this.</summary>
        public static TzsError PathNotFound(string path) {
            return new TzsError("E_PATH_NOT_FOUND", "路径 不存在: " + path);
        }

        /// <summary>The designer's own rules refused it (repeat gating, duplicate name, cited).
        /// The caller must not retry; this is a fact about the form, not about the request.</summary>
        public static TzsError Designer(string message) {
            return new TzsError("designer", message);
        }
    }
}
