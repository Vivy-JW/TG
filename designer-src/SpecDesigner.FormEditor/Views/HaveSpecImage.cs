using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media.Imaging;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x0200003C RID: 60
	public class HaveSpecImage : Image
	{
		// Token: 0x0600021F RID: 543 RVA: 0x0000BB08 File Offset: 0x00009D08
		public HaveSpecImage()
		{
			base.Width = 12.0;
			base.Height = 12.0;
			base.HorizontalAlignment = HorizontalAlignment.Left;
			base.Visibility = Visibility.Collapsed;
		}

		// Token: 0x17000069 RID: 105
		// (get) Token: 0x06000220 RID: 544 RVA: 0x0000BB3C File Offset: 0x00009D3C
		// (set) Token: 0x06000221 RID: 545 RVA: 0x0000BB4E File Offset: 0x00009D4E
		public SpecStatus SpecNodeStatus
		{
			get
			{
				return (SpecStatus)base.GetValue(HaveSpecImage.SpecNodeStatusProperty);
			}
			set
			{
				base.SetValue(HaveSpecImage.SpecNodeStatusProperty, value);
			}
		}

		// Token: 0x1700006A RID: 106
		// (get) Token: 0x06000222 RID: 546 RVA: 0x0000BB61 File Offset: 0x00009D61
		// (set) Token: 0x06000223 RID: 547 RVA: 0x0000BB73 File Offset: 0x00009D73
		public bool SpecificationDefined
		{
			get
			{
				return (bool)base.GetValue(HaveSpecImage.SpecificationDefinedProperty);
			}
			set
			{
				base.SetValue(HaveSpecImage.SpecificationDefinedProperty, value);
			}
		}

		// Token: 0x06000224 RID: 548 RVA: 0x0000BB88 File Offset: 0x00009D88
		public static void OnStatusPropertyChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
		{
			HaveSpecImage haveSpecImage = d as HaveSpecImage;
			if (haveSpecImage == null)
			{
				return;
			}
			bool flag = (haveSpecImage.SpecNodeStatus & SpecStatus.MODIFY) == SpecStatus.MODIFY;
			BitmapImage bitmapImage = null;
			if (flag && haveSpecImage.SpecificationDefined)
			{
				bitmapImage = HaveSpecImage.HasModifyAndSpec;
			}
			else if (!flag && haveSpecImage.SpecificationDefined)
			{
				bitmapImage = HaveSpecImage.HasSpec;
			}
			else if (flag && !haveSpecImage.SpecificationDefined)
			{
				bitmapImage = HaveSpecImage.HasModify;
			}
			if (bitmapImage != null)
			{
				haveSpecImage.Source = bitmapImage;
				haveSpecImage.Visibility = Visibility.Visible;
				return;
			}
			haveSpecImage.Visibility = Visibility.Collapsed;
		}

		// Token: 0x0400012C RID: 300
		public static readonly BitmapImage HasSpec = new BitmapImage(new Uri("pack://application:,,,/SpecDesigner.FormEditor;component/Images/icon_sd.png"));

		// Token: 0x0400012D RID: 301
		public static readonly BitmapImage HasModify = new BitmapImage(new Uri("pack://application:,,,/SpecDesigner.FormEditor;component/Images/icon_pr.png"));

		// Token: 0x0400012E RID: 302
		public static readonly BitmapImage HasModifyAndSpec = new BitmapImage(new Uri("pack://application:,,,/SpecDesigner.FormEditor;component/Images/icon_prsd.png"));

		// Token: 0x0400012F RID: 303
		public static readonly DependencyProperty SpecNodeStatusProperty = DependencyProperty.Register("SpecNodeStatus", typeof(SpecStatus), typeof(HaveSpecImage), new PropertyMetadata(SpecStatus.NULL, new PropertyChangedCallback(HaveSpecImage.OnStatusPropertyChanged)));

		// Token: 0x04000130 RID: 304
		public static readonly DependencyProperty SpecificationDefinedProperty = DependencyProperty.Register("SpecificationDefined", typeof(bool), typeof(HaveSpecImage), new PropertyMetadata(false, new PropertyChangedCallback(HaveSpecImage.OnStatusPropertyChanged)));
	}
}
