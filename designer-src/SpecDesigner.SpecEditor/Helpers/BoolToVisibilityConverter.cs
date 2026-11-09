using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000023 RID: 35
	public class BoolToVisibilityConverter : IValueConverter
	{
		// Token: 0x1700001F RID: 31
		// (get) Token: 0x06000104 RID: 260 RVA: 0x000095C3 File Offset: 0x000077C3
		// (set) Token: 0x06000105 RID: 261 RVA: 0x000095CB File Offset: 0x000077CB
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

		// Token: 0x06000106 RID: 262 RVA: 0x000095D4 File Offset: 0x000077D4
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
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

		// Token: 0x06000107 RID: 263 RVA: 0x00009614 File Offset: 0x00007814
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
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

		// Token: 0x06000108 RID: 264 RVA: 0x00009660 File Offset: 0x00007860
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

		// Token: 0x04000090 RID: 144
		private Visibility _falseToVisibility = Visibility.Collapsed;
	}
}
