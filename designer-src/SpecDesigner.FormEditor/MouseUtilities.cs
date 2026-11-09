using System;
using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Interop;
using System.Windows.Media;

namespace SpecDesigner.FormEditor
{
	// Token: 0x02000007 RID: 7
	public class MouseUtilities
	{
		// Token: 0x06000020 RID: 32
		[DllImport("user32.dll")]
		private static extern bool GetCursorPos(ref MouseUtilities.Win32Point pt);

		// Token: 0x06000021 RID: 33
		[DllImport("user32.dll")]
		private static extern bool ScreenToClient(IntPtr hwnd, ref MouseUtilities.Win32Point pt);

		// Token: 0x06000022 RID: 34 RVA: 0x000027F8 File Offset: 0x000009F8
		public static Point GetMousePosition(Visual relativeTo)
		{
			MouseUtilities.Win32Point win32Point = default(MouseUtilities.Win32Point);
			MouseUtilities.GetCursorPos(ref win32Point);
			HwndSource hwndSource = (HwndSource)PresentationSource.FromVisual(relativeTo);
			MouseUtilities.ScreenToClient(hwndSource.Handle, ref win32Point);
			GeneralTransform generalTransform = relativeTo.TransformToAncestor(hwndSource.RootVisual);
			Point point = generalTransform.Transform(new Point(0.0, 0.0));
			return new Point((double)win32Point.X - point.X, (double)win32Point.Y - point.Y);
		}

		// Token: 0x02000008 RID: 8
		private struct Win32Point
		{
			// Token: 0x04000013 RID: 19
			public int X;

			// Token: 0x04000014 RID: 20
			public int Y;
		}
	}
}
