using System;
using System.ComponentModel;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Documents;
using System.Windows.Media;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x0200003A RID: 58
	public class WidgetRepeatAdorner : Adorner, IDisposable
	{
		// Token: 0x06000210 RID: 528 RVA: 0x0000B28C File Offset: 0x0000948C
		public WidgetRepeatAdorner(FrameworkElement ui)
			: base(ui)
		{
			Binding binding = new Binding("IsVisible");
			binding.Source = ui;
			binding.Converter = new BooleanToVisibilityConverter();
			base.SetBinding(UIElement.VisibilityProperty, binding);
			this._source = ui.DataContext as XmlElement;
			base.IsHitTestVisible = false;
			if (this._source != null)
			{
				this._source.PropertyChanged += this._source_PropertyChanged;
			}
		}

		// Token: 0x06000211 RID: 529 RVA: 0x0000B304 File Offset: 0x00009504
		private void _source_PropertyChanged(object sender, PropertyChangedEventArgs e)
		{
			string propertyName;
			switch (propertyName = e.PropertyName)
			{
			case "":
				base.InvalidateVisual();
				break;
			case "repeat":
			case "stepX":
			case "stepY":
			case "columnCount":
			case "rowCount":
				break;

				return;
			}
		}

		// Token: 0x06000212 RID: 530 RVA: 0x0000B3B4 File Offset: 0x000095B4
		protected override void OnRender(DrawingContext drawingContext)
		{
			base.OnRender(drawingContext);
			if (base.Visibility != Visibility.Visible)
			{
				return;
			}
			string attribute = this._source.GetAttribute("repeat");
			if (attribute != null && attribute == "true")
			{
				string text = this._source.LocalString;
				if (text == null)
				{
					text = this._source.Name;
				}
				FormattedText formattedText = new FormattedText(text, CultureInfo.CurrentCulture, FlowDirection.LeftToRight, new Typeface(SystemFonts.CaptionFontFamily.Source), 12.0, Brushes.DarkBlue);
				int rowCount = this._source.RowCount;
				int columnCount = this._source.ColumnCount;
				int num = 0;
				int num2 = 0;
				Pen pen = new Pen(Brushes.Gray, 1.0);
				for (int i = 0; i < rowCount; i++)
				{
					num2 = i * (this._source.Height + this._source.StepY * FormDesignSetting.UnitHeight);
					num = 0;
					for (int j = 0; j < columnCount; j++)
					{
						if (i != 0 || j != 0)
						{
							num = j * (this._source.Width + this._source.StepX * FormDesignSetting.UnitWidth);
							Rect rect = new Rect((double)num, (double)num2, (double)this._source.Width, (double)this._source.Height);
							drawingContext.DrawRectangle(Brushes.Gainsboro, pen, rect);
							drawingContext.DrawText(formattedText, new Point((double)num, (double)num2));
						}
					}
				}
				if (rowCount + columnCount > 1)
				{
					num += this._source.Width;
					num2 += this._source.Height;
					Rect rect2 = new Rect(new Point(-1.0, -1.0), new Point((double)(num + 1), (double)(num2 + 1)));
					Pen pen2 = new Pen(Brushes.WhiteSmoke, 1.0);
					drawingContext.DrawRectangle(null, pen2, rect2);
				}
			}
		}

		// Token: 0x06000213 RID: 531 RVA: 0x0000B5A1 File Offset: 0x000097A1
		public void Dispose()
		{
			if (this._source != null)
			{
				this._source.PropertyChanged -= this._source_PropertyChanged;
			}
		}

		// Token: 0x04000126 RID: 294
		private XmlElement _source;
	}
}
