using System;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000033 RID: 51
	public class ColumnCanvas : Canvas
	{
		// Token: 0x17000060 RID: 96
		// (get) Token: 0x060001DE RID: 478 RVA: 0x00009BEC File Offset: 0x00007DEC
		// (set) Token: 0x060001DF RID: 479 RVA: 0x00009BFE File Offset: 0x00007DFE
		public BitmapImage TreeImage
		{
			get
			{
				return (BitmapImage)base.GetValue(ColumnCanvas.TreeImageProperty);
			}
			set
			{
				base.SetValue(ColumnCanvas.TreeImageProperty, value);
			}
		}

		// Token: 0x17000061 RID: 97
		// (get) Token: 0x060001E0 RID: 480 RVA: 0x00009C0C File Offset: 0x00007E0C
		// (set) Token: 0x060001E1 RID: 481 RVA: 0x00009C1E File Offset: 0x00007E1E
		public Brush ColumnBrush
		{
			get
			{
				return (Brush)base.GetValue(ColumnCanvas.ColumnBrushProperty);
			}
			set
			{
				base.SetValue(ColumnCanvas.ColumnBrushProperty, value);
			}
		}

		// Token: 0x17000062 RID: 98
		// (get) Token: 0x060001E2 RID: 482 RVA: 0x00009C2C File Offset: 0x00007E2C
		// (set) Token: 0x060001E3 RID: 483 RVA: 0x00009C3E File Offset: 0x00007E3E
		public string AggregateText
		{
			get
			{
				return (string)base.GetValue(ColumnCanvas.AggregateTextProperty);
			}
			set
			{
				base.SetValue(ColumnCanvas.AggregateTextProperty, value);
			}
		}

		// Token: 0x17000063 RID: 99
		// (get) Token: 0x060001E4 RID: 484 RVA: 0x00009C4C File Offset: 0x00007E4C
		// (set) Token: 0x060001E5 RID: 485 RVA: 0x00009C5E File Offset: 0x00007E5E
		public int TotalRows
		{
			get
			{
				return (int)base.GetValue(ColumnCanvas.TotalRowsProperty);
			}
			set
			{
				base.SetValue(ColumnCanvas.TotalRowsProperty, value);
			}
		}

		// Token: 0x17000064 RID: 100
		// (get) Token: 0x060001E6 RID: 486 RVA: 0x00009C71 File Offset: 0x00007E71
		// (set) Token: 0x060001E7 RID: 487 RVA: 0x00009C83 File Offset: 0x00007E83
		public int RowHeight
		{
			get
			{
				return (int)base.GetValue(ColumnCanvas.RowHeightProperty);
			}
			set
			{
				base.SetValue(ColumnCanvas.RowHeightProperty, value);
			}
		}

		// Token: 0x17000065 RID: 101
		// (get) Token: 0x060001E8 RID: 488 RVA: 0x00009C96 File Offset: 0x00007E96
		// (set) Token: 0x060001E9 RID: 489 RVA: 0x00009CA8 File Offset: 0x00007EA8
		public bool IsEnabledAggregate
		{
			get
			{
				return (bool)base.GetValue(ColumnCanvas.IsEnabledAggregateProperty);
			}
			set
			{
				base.SetValue(ColumnCanvas.IsEnabledAggregateProperty, value);
			}
		}

		// Token: 0x17000066 RID: 102
		// (get) Token: 0x060001EA RID: 490 RVA: 0x00009CBB File Offset: 0x00007EBB
		// (set) Token: 0x060001EB RID: 491 RVA: 0x00009CCD File Offset: 0x00007ECD
		public bool IsAnySiblingEnabledAggregate
		{
			get
			{
				return (bool)base.GetValue(ColumnCanvas.IsAnySiblingEnabledAggregateProperty);
			}
			set
			{
				base.SetValue(ColumnCanvas.IsAnySiblingEnabledAggregateProperty, value);
			}
		}

		// Token: 0x060001EC RID: 492 RVA: 0x00009CE0 File Offset: 0x00007EE0
		public static void OnBindingPropertyChanged(DependencyObject obj, DependencyPropertyChangedEventArgs e)
		{
			FrameworkElement frameworkElement = obj as FrameworkElement;
			frameworkElement.InvalidateVisual();
		}

		// Token: 0x17000067 RID: 103
		// (get) Token: 0x060001EE RID: 494 RVA: 0x00009D04 File Offset: 0x00007F04
		private string CellText
		{
			get
			{
				if (this._source == null)
				{
					this._source = base.DataContext as XmlElement;
				}
				string empty = string.Empty;
				switch (FormSpecModel.GetNodeType(this._source))
				{
				case SpecNodeType.PROGREL:
					return "ProgQuery";
				case SpecNodeType.REFERENCE:
					return "Reference";
				case SpecNodeType.MULTILANG:
					return "MultiLang";
				}
				return this._source.Type.ToString();
			}
		}

		// Token: 0x060001EF RID: 495 RVA: 0x00009D88 File Offset: 0x00007F88
		protected override void OnRender(DrawingContext drawingContext)
		{
			base.OnRender(drawingContext);
			string cellText = this.CellText;
			if (this._source == null || this._source.Parent == null)
			{
				return;
			}
			if (this.ColumnBrush == null)
			{
				return;
			}
			Pen pen = new Pen(this.ColumnBrush, 1.0);
			FormattedText formattedText = new FormattedText(cellText, CultureInfo.CurrentCulture, FlowDirection.LeftToRight, new Typeface(SystemFonts.CaptionFontFamily.Source), 12.0, Brushes.Black);
			int num = 0;
			int num2 = 2;
			int num3 = this._source.Parent.RowHeight * FormDesignSetting.UnitHeight;
			for (int i = 0; i < this.TotalRows; i++)
			{
				Rect rect = new Rect(0.0, (double)num, (double)this._source.Width, (double)num3);
				switch (this._source.Parent.Type)
				{
				case ComponentType.Table:
					if (i > 0)
					{
						drawingContext.DrawLine(pen, new Point(0.0, rect.Y), new Point((double)this._source.Width, rect.Y));
					}
					drawingContext.DrawText(formattedText, new Point((double)num2, (double)num));
					break;
				case ComponentType.Tree:
					if (this._source.Index == 0)
					{
						switch (i % 4)
						{
						case 0:
							if (this.TreeImage != null)
							{
								drawingContext.DrawImage(this.TreeImage, new Rect(new Point((double)num2, (double)(num + 5)), new Point((double)(num2 + 9), (double)(num + 9 + 5))));
							}
							drawingContext.DrawText(formattedText, new Point((double)(num2 + 10), (double)num));
							break;
						case 1:
							if (this.TreeImage != null)
							{
								drawingContext.DrawImage(this.TreeImage, new Rect(new Point((double)(num2 + 20), (double)(num + 5)), new Point((double)(num2 + 29), (double)(num + 9 + 5))));
							}
							drawingContext.DrawText(formattedText, new Point((double)(num2 + 30), (double)num));
							break;
						default:
							drawingContext.DrawText(formattedText, new Point((double)(num2 + 40), (double)num));
							break;
						}
					}
					else
					{
						drawingContext.DrawText(formattedText, new Point((double)num2, (double)num));
					}
					break;
				}
				num += num3;
			}
			if (this.IsEnabledAggregate)
			{
				Rect rect2 = new Rect(0.0, (double)num, (double)this._source.Width, (double)num3);
				drawingContext.DrawLine(pen, new Point(0.0, rect2.Y), new Point((double)this._source.Width, rect2.Y));
				formattedText = new FormattedText(this.AggregateText, CultureInfo.CurrentCulture, FlowDirection.LeftToRight, new Typeface(SystemFonts.CaptionFontFamily.Source), 12.0, Brushes.Black);
				drawingContext.DrawRectangle(this.ColumnBrush, pen, rect2);
				drawingContext.DrawText(formattedText, new Point((double)num2, (double)num));
			}
		}

		// Token: 0x0400010D RID: 269
		public static readonly DependencyProperty TreeImageProperty = DependencyProperty.Register("TreeImage", typeof(BitmapImage), typeof(ColumnCanvas), new PropertyMetadata(null));

		// Token: 0x0400010E RID: 270
		public static readonly DependencyProperty ColumnBrushProperty = DependencyProperty.Register("ColumnBrus", typeof(Brush), typeof(ColumnCanvas), new PropertyMetadata(null));

		// Token: 0x0400010F RID: 271
		public static readonly DependencyProperty AggregateTextProperty = DependencyProperty.Register("AggregateText", typeof(string), typeof(ColumnCanvas), new PropertyMetadata(string.Empty, new PropertyChangedCallback(ColumnCanvas.OnBindingPropertyChanged)));

		// Token: 0x04000110 RID: 272
		public static readonly DependencyProperty TotalRowsProperty = DependencyProperty.Register("TotalRows", typeof(int), typeof(ColumnCanvas), new PropertyMetadata(new PropertyChangedCallback(ColumnCanvas.OnBindingPropertyChanged)));

		// Token: 0x04000111 RID: 273
		public static readonly DependencyProperty RowHeightProperty = DependencyProperty.Register("RowHeight", typeof(int), typeof(ColumnCanvas), new PropertyMetadata(new PropertyChangedCallback(ColumnCanvas.OnBindingPropertyChanged)));

		// Token: 0x04000112 RID: 274
		public static readonly DependencyProperty IsEnabledAggregateProperty = DependencyProperty.Register("IsEnabledAggregate", typeof(bool), typeof(ColumnCanvas), new PropertyMetadata(false, new PropertyChangedCallback(ColumnCanvas.OnBindingPropertyChanged)));

		// Token: 0x04000113 RID: 275
		public static readonly DependencyProperty IsAnySiblingEnabledAggregateProperty = DependencyProperty.Register("IsAnySiblingEnabledAggregate", typeof(bool), typeof(ColumnCanvas), new PropertyMetadata(false, new PropertyChangedCallback(ColumnCanvas.OnBindingPropertyChanged)));

		// Token: 0x04000114 RID: 276
		private XmlElement _source;
	}
}
