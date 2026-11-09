using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000024 RID: 36
	public class VisibilityToBoolConverter : IValueConverter
	{
		// Token: 0x17000020 RID: 32
		// (get) Token: 0x0600010A RID: 266 RVA: 0x0000969B File Offset: 0x0000789B
		// (set) Token: 0x0600010B RID: 267 RVA: 0x000096A3 File Offset: 0x000078A3
		public Visibility FalseToVisibility
		{
			get
			{
				return this._falseToVisibility;
			}
			set
			{
				this._falseToVisibility = value;
			}
		}

		// Token: 0x0600010C RID: 268 RVA: 0x000096AC File Offset: 0x000078AC
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			object obj;
			try
			{
				bool flag = (Visibility)value != this.FalseToVisibility;
				obj = flag;
			}
			catch
			{
				throw;
			}
			return obj;
		}

		// Token: 0x0600010D RID: 269 RVA: 0x000096EC File Offset: 0x000078EC
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			object obj;
			try
			{
				Visibility visibility;
				if ((bool)value)
				{
					visibility = this.ReverseVisibility(this.FalseToVisibility);
				}
				else
				{
					visibility = this.FalseToVisibility;
				}
				obj = visibility;
			}
			catch
			{
				throw;
			}
			return obj;
		}

		// Token: 0x0600010E RID: 270 RVA: 0x00009738 File Offset: 0x00007938
		private Visibility ReverseVisibility(Visibility vsi)
		{
			Visibility visibility = Visibility.Visible;
			switch (vsi)
			{
			case Visibility.Visible:
				visibility = Visibility.Collapsed;
				break;
			case Visibility.Collapsed:
				visibility = Visibility.Visible;
				break;
			}
			return visibility;
		}

		// Token: 0x04000091 RID: 145
		private Visibility _falseToVisibility = Visibility.Collapsed;
	}
}
