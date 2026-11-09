using System;
using System.Windows;
using System.Windows.Media;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200000F RID: 15
	public static class Visual_ExtensionMethods
	{
		// Token: 0x0600006A RID: 106 RVA: 0x000063D0 File Offset: 0x000045D0
		public static T SafeFindDescendant<T>(this Visual @this, Predicate<T> predicate = null) where T : Visual
		{
			T t = default(T);
			if (@this == null)
			{
				return default(T);
			}
			int childrenCount = VisualTreeHelper.GetChildrenCount(@this);
			for (int i = 0; i < childrenCount; i++)
			{
				DependencyObject child = VisualTreeHelper.GetChild(@this, i);
				T t2 = child as T;
				if (t2 == null)
				{
					t = ((Visual)child).SafeFindDescendant<T>(predicate);
					if (t != null)
					{
						break;
					}
				}
				else if (predicate == null || predicate(t2))
				{
					t = t2;
					break;
				}
			}
			return t;
		}

		// Token: 0x0600006B RID: 107 RVA: 0x00006450 File Offset: 0x00004650
		public static T SafeFindAncestor<T>(this Visual @this, Predicate<T> predicate = null) where T : Visual
		{
			if (@this == null)
			{
				return default(T);
			}
			DependencyObject parent = VisualTreeHelper.GetParent(@this);
			T t = parent as T;
			if (t != null && (predicate == null || predicate(t)))
			{
				return t;
			}
			return ((Visual)parent).SafeFindAncestor<T>(predicate);
		}
	}
}
