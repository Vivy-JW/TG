using System;
using System.Runtime.InteropServices;
using System.Text;

namespace SpecDesignerCommon.Site.ViewModels
{
	// Token: 0x02000134 RID: 308
	[Guid("000214F9-0000-0000-C000-000000000046")]
	[InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
	[ComImport]
	internal interface IShellLink
	{
		// Token: 0x06000AE2 RID: 2786
		void GetPath([MarshalAs(UnmanagedType.LPWStr)] [Out] StringBuilder pszFile, int cchMaxPath, out IntPtr pfd, int fFlags);

		// Token: 0x06000AE3 RID: 2787
		void GetIDList(out IntPtr ppidl);

		// Token: 0x06000AE4 RID: 2788
		void SetIDList(IntPtr pidl);

		// Token: 0x06000AE5 RID: 2789
		void GetDescription([MarshalAs(UnmanagedType.LPWStr)] [Out] StringBuilder pszName, int cchMaxName);

		// Token: 0x06000AE6 RID: 2790
		void SetDescription([MarshalAs(UnmanagedType.LPWStr)] string pszName);

		// Token: 0x06000AE7 RID: 2791
		void GetWorkingDirectory([MarshalAs(UnmanagedType.LPWStr)] [Out] StringBuilder pszDir, int cchMaxPath);

		// Token: 0x06000AE8 RID: 2792
		void SetWorkingDirectory([MarshalAs(UnmanagedType.LPWStr)] string pszDir);

		// Token: 0x06000AE9 RID: 2793
		void GetArguments([MarshalAs(UnmanagedType.LPWStr)] [Out] StringBuilder pszArgs, int cchMaxPath);

		// Token: 0x06000AEA RID: 2794
		void SetArguments([MarshalAs(UnmanagedType.LPWStr)] string pszArgs);

		// Token: 0x06000AEB RID: 2795
		void GetHotkey(out short pwHotkey);

		// Token: 0x06000AEC RID: 2796
		void SetHotkey(short wHotkey);

		// Token: 0x06000AED RID: 2797
		void GetShowCmd(out int piShowCmd);

		// Token: 0x06000AEE RID: 2798
		void SetShowCmd(int iShowCmd);

		// Token: 0x06000AEF RID: 2799
		void GetIconLocation([MarshalAs(UnmanagedType.LPWStr)] [Out] StringBuilder pszIconPath, int cchIconPath, out int piIcon);

		// Token: 0x06000AF0 RID: 2800
		void SetIconLocation([MarshalAs(UnmanagedType.LPWStr)] string pszIconPath, int iIcon);

		// Token: 0x06000AF1 RID: 2801
		void SetRelativePath([MarshalAs(UnmanagedType.LPWStr)] string pszPathRel, int dwReserved);

		// Token: 0x06000AF2 RID: 2802
		void Resolve(IntPtr hwnd, int fFlags);

		// Token: 0x06000AF3 RID: 2803
		void SetPath([MarshalAs(UnmanagedType.LPWStr)] string pszFile);
	}
}
