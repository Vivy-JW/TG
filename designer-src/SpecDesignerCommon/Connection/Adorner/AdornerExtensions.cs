using System;
using System.Windows;
using System.Windows.Documents;

namespace SpecDesignerCommon.Connection.Adorner
{
	// Token: 0x0200006C RID: 108
	public static class AdornerExtensions
	{
		// Token: 0x0600040D RID: 1037 RVA: 0x00012B50 File Offset: 0x00010D50
		public static void TryRemoveAdorners<T>(this UIElement elem) where T : Adorner
		{
			AdornerLayer adornerLayer = AdornerLayer.GetAdornerLayer(elem);
			if (adornerLayer != null)
			{
				adornerLayer.RemoveAdorners<T>(elem);
			}
		}

		// Token: 0x0600040E RID: 1038 RVA: 0x00012B70 File Offset: 0x00010D70
		public static void RemoveAdorners<T>(this AdornerLayer adr, UIElement elem) where T : Adorner
		{
			Adorner[] adorners = adr.GetAdorners(elem);
			if (adorners == null)
			{
				return;
			}
			for (int i = adorners.Length - 1; i >= 0; i--)
			{
				if (adorners[i] is T)
				{
					adr.Remove(adorners[i]);
				}
			}
		}

		// Token: 0x0600040F RID: 1039 RVA: 0x00012BAC File Offset: 0x00010DAC
		public static void TryAddAdorner<T>(this UIElement elem, Adorner adorner) where T : Adorner
		{
			AdornerLayer adornerLayer = AdornerLayer.GetAdornerLayer(elem);
			if (adornerLayer != null && !adornerLayer.ContainsAdorner<T>(elem))
			{
				adornerLayer.Add(adorner);
			}
		}

		// Token: 0x06000410 RID: 1040 RVA: 0x00012BD4 File Offset: 0x00010DD4
		public static bool ContainsAdorner<T>(this AdornerLayer adr, UIElement elem) where T : Adorner
		{
			Adorner[] adorners = adr.GetAdorners(elem);
			if (adorners == null)
			{
				return false;
			}
			for (int i = adorners.Length - 1; i >= 0; i--)
			{
				if (adorners[i] is T)
				{
					return true;
				}
			}
			return false;
		}

		// Token: 0x06000411 RID: 1041 RVA: 0x00012C0C File Offset: 0x00010E0C
		public static void RemoveAllAdorners(this AdornerLayer adr, UIElement elem)
		{
			Adorner[] adorners = adr.GetAdorners(elem);
			if (adorners == null)
			{
				return;
			}
			foreach (Adorner adorner in adorners)
			{
				adr.Remove(adorner);
			}
		}
	}
}
