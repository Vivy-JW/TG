using System;
using System.Collections.Generic;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Media;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000004 RID: 4
	public static class VisualTreeHelperEx
	{
		// Token: 0x0600000E RID: 14 RVA: 0x0000233C File Offset: 0x0000053C
		public static T FindVisualParent<T>(DependencyObject child) where T : DependencyObject
		{
			DependencyObject parent = VisualTreeHelper.GetParent(child);
			if (parent == null)
			{
				return default(T);
			}
			T t = parent as T;
			if (t != null)
			{
				return t;
			}
			return VisualTreeHelperEx.FindVisualParent<T>(parent);
		}

		// Token: 0x0600000F RID: 15 RVA: 0x000023B0 File Offset: 0x000005B0
		public static T GetParentByPoint<T>(UIElement container, Point pt) where T : class
		{
			T r = default(T);
			VisualTreeHelper.HitTest(container, pt);
			VisualTreeHelper.HitTest(container, null, delegate(HitTestResult result)
			{
				T t = VisualTreeHelperEx.FindVisualParent1<T>(result.VisualHit);
				if (t != null)
				{
					r = t;
					return HitTestResultBehavior.Stop;
				}
				return HitTestResultBehavior.Continue;
			}, new PointHitTestParameters(pt));
			return r;
		}

		// Token: 0x06000010 RID: 16 RVA: 0x000023F8 File Offset: 0x000005F8
		public static T FindVisualParent1<T>(DependencyObject child) where T : class
		{
			DependencyObject parent = VisualTreeHelper.GetParent(child);
			if (parent == null)
			{
				return default(T);
			}
			T t = parent as T;
			if (t != null)
			{
				return t;
			}
			return VisualTreeHelperEx.FindVisualParent1<T>(parent);
		}

		// Token: 0x06000011 RID: 17 RVA: 0x00002438 File Offset: 0x00000638
		public static T FindLogicVisualParent1<T>(DependencyObject child) where T : class
		{
			DependencyObject parent = LogicalTreeHelper.GetParent(child);
			if (parent == null)
			{
				return default(T);
			}
			T t = parent as T;
			if (t != null)
			{
				return t;
			}
			return VisualTreeHelperEx.FindLogicVisualParent1<T>(parent);
		}

		// Token: 0x06000012 RID: 18 RVA: 0x00002478 File Offset: 0x00000678
		public static T FindChild<T>(DependencyObject parent) where T : class
		{
			if (parent == null)
			{
				return default(T);
			}
			T t = default(T);
			int childrenCount = VisualTreeHelper.GetChildrenCount(parent);
			for (int i = 0; i < childrenCount; i++)
			{
				DependencyObject child = VisualTreeHelper.GetChild(parent, i);
				T t2 = child as T;
				if (t2 != null)
				{
					return t2;
				}
				t = VisualTreeHelperEx.FindChild<T>(child);
				if (t != null)
				{
					break;
				}
			}
			return t;
		}

		// Token: 0x06000013 RID: 19 RVA: 0x000024E4 File Offset: 0x000006E4
		public static void FindAllChild<T>(FrameworkElement parent, List<T> folders) where T : class
		{
			if (parent == null)
			{
				return;
			}
			T t = default(T);
			int childrenCount = VisualTreeHelper.GetChildrenCount(parent);
			for (int i = 0; i < childrenCount; i++)
			{
				DependencyObject child = VisualTreeHelper.GetChild(parent, i);
				T t2 = child as T;
				if (t2 == null)
				{
					VisualTreeHelperEx.FindAllChild<T>((FrameworkElement)child, folders);
					if (t != null)
					{
						return;
					}
				}
				else
				{
					folders.Add(t2);
					VisualTreeHelperEx.FindAllChild<T>((FrameworkElement)child, folders);
				}
			}
		}

		// Token: 0x06000014 RID: 20 RVA: 0x00002558 File Offset: 0x00000758
		public static DataGridColumnHeader GetColumnHeaderFromColumn(DataGridColumn column, DataGrid dataGrid)
		{
			List<DataGridColumnHeader> visualChildCollection = VisualTreeHelperEx.GetVisualChildCollection<DataGridColumnHeader>(dataGrid);
			foreach (DataGridColumnHeader dataGridColumnHeader in visualChildCollection)
			{
				if (dataGridColumnHeader.Column == column)
				{
					return dataGridColumnHeader;
				}
			}
			return null;
		}

		// Token: 0x06000015 RID: 21 RVA: 0x000025B8 File Offset: 0x000007B8
		public static List<T> GetVisualChildCollection<T>(object parent) where T : Visual
		{
			List<T> list = new List<T>();
			VisualTreeHelperEx.GetVisualChildCollection<T>(parent as DependencyObject, list);
			return list;
		}

		// Token: 0x06000016 RID: 22 RVA: 0x000025D8 File Offset: 0x000007D8
		private static void GetVisualChildCollection<T>(DependencyObject parent, List<T> visualCollection) where T : Visual
		{
			int childrenCount = VisualTreeHelper.GetChildrenCount(parent);
			for (int i = 0; i < childrenCount; i++)
			{
				DependencyObject child = VisualTreeHelper.GetChild(parent, i);
				if (child is T)
				{
					visualCollection.Add(child as T);
				}
				else if (child != null)
				{
					VisualTreeHelperEx.GetVisualChildCollection<T>(child, visualCollection);
				}
			}
		}
	}
}
