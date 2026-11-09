using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Markup;
using SpecDesigner.FormEditor.Petzold.Media2D;

namespace SpecDesigner.FormEditor.Relation
{
	// Token: 0x0200003F RID: 63
	public partial class TBLRelation : UserControl
	{
		// Token: 0x0600022E RID: 558 RVA: 0x0000BDC8 File Offset: 0x00009FC8
		public TBLRelation()
		{
			this.InitializeComponent();
		}

		// Token: 0x1700006B RID: 107
		// (get) Token: 0x0600022F RID: 559 RVA: 0x0000BDD6 File Offset: 0x00009FD6
		// (set) Token: 0x06000230 RID: 560 RVA: 0x0000BDE0 File Offset: 0x00009FE0
		public TBLControl SourceController
		{
			get
			{
				return this.sourceControl;
			}
			set
			{
				this.sourceControl = value;
				base.SetBinding(TBLRelation.SourceProperty, new Binding
				{
					Source = value,
					Path = new PropertyPath(TBLControl.FootAnchorPointProperty)
				});
			}
		}

		// Token: 0x1700006C RID: 108
		// (get) Token: 0x06000231 RID: 561 RVA: 0x0000BE1E File Offset: 0x0000A01E
		// (set) Token: 0x06000232 RID: 562 RVA: 0x0000BE28 File Offset: 0x0000A028
		public TBLControl TargetController
		{
			get
			{
				return this.targetControl;
			}
			set
			{
				this.targetControl = value;
				base.SetBinding(TBLRelation.DestinationProperty, new Binding
				{
					Source = value,
					Path = new PropertyPath(TBLControl.HeaderAnchorPointProperty)
				});
			}
		}

		// Token: 0x1700006D RID: 109
		// (get) Token: 0x06000233 RID: 563 RVA: 0x0000BE66 File Offset: 0x0000A066
		// (set) Token: 0x06000234 RID: 564 RVA: 0x0000BE78 File Offset: 0x0000A078
		public Point Source
		{
			get
			{
				return (Point)base.GetValue(TBLRelation.SourceProperty);
			}
			set
			{
				base.SetValue(TBLRelation.SourceProperty, value);
			}
		}

		// Token: 0x1700006E RID: 110
		// (get) Token: 0x06000235 RID: 565 RVA: 0x0000BE8B File Offset: 0x0000A08B
		// (set) Token: 0x06000236 RID: 566 RVA: 0x0000BE9D File Offset: 0x0000A09D
		public Point Destination
		{
			get
			{
				return (Point)base.GetValue(TBLRelation.DestinationProperty);
			}
			set
			{
				base.SetValue(TBLRelation.DestinationProperty, value);
			}
		}

		// Token: 0x04000132 RID: 306
		private TBLControl sourceControl;

		// Token: 0x04000133 RID: 307
		private TBLControl targetControl;

		// Token: 0x04000134 RID: 308
		public static readonly DependencyProperty SourceProperty = DependencyProperty.Register("Source", typeof(Point), typeof(TBLRelation), new FrameworkPropertyMetadata(default(Point)));

		// Token: 0x04000135 RID: 309
		public static readonly DependencyProperty DestinationProperty = DependencyProperty.Register("Destination", typeof(Point), typeof(TBLRelation), new FrameworkPropertyMetadata(default(Point)));
	}
}
