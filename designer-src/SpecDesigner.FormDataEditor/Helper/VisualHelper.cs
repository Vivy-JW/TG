using System;
using System.Windows.Media;

namespace SpecDesigner.FormDataEditor.Helper
{
	// Token: 0x02000005 RID: 5
	public class VisualHelper
	{
		// Token: 0x0600000B RID: 11 RVA: 0x00002148 File Offset: 0x00000348
		public static T GetVisualChild<T>(Visual parent) where T : Visual
		{
			T t = default(T);
			int childrenCount = VisualTreeHelper.GetChildrenCount(parent);
			for (int i = 0; i < childrenCount; i++)
			{
				Visual visual = (Visual)VisualTreeHelper.GetChild(parent, i);
				t = visual as T;
				if (t == null)
				{
					t = VisualHelper.GetVisualChild<T>(visual);
				}
				if (t != null)
				{
					break;
				}
			}
			return t;
		}
	}
}
